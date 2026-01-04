/*
@Author  :   xff
@Desc    :
*/

package middleware

import (
	"errors"
	"goInsight/global"
	userModels "goInsight/internal/users/models"
	"goInsight/pkg/ldap"

	"goInsight/pkg/response"

	"github.com/gin-contrib/requestid"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// login form
type login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
	OtpCode  string `form:"otp_code" json:"otp_code"`
}

// user info
type userInfo struct {
	username string
	password string
	user     *userModels.InsightUsers
}

func (u *userInfo) checkIfUserExist() (err error) {
	result := global.App.DB.Table("insight_users u").
		Where("u.username=?", u.username).
		Scan(&u.user)
	if result.RowsAffected == 0 {
		return errors.New("用户名或密码错误")
	}
	return nil
}

func (u *userInfo) checkUserValid() error {
	// 用户是否激活
	if !u.user.IsActive {
		return errors.New("用户未激活")
	}
	// 验证密码
	err := bcrypt.CompareHashAndPassword([]byte(u.user.Password), []byte(u.password))
	if err != nil {
		return errors.New("incorrect Username or Password")
	}
	return nil
}

func (u *userInfo) checkIfUserNeedsOTP() bool {
	// 用户是否开启2FA认证
	return u.user.IsTwoFA
}

func (u *userInfo) checkIfOtpSecretNullString() bool {
	// 检查用户的otp秘钥是否为空
	return len(u.user.OtpSecret) == 0
}

func OTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// login json bind to loginVals
		var loginVals login
		if err := c.ShouldBind(&loginVals); err != nil {
			response.Fail(c, "missing Username or Password")
			return
		}
		username := loginVals.Username
		password := loginVals.Password

		// LDAP 认证
		if global.App.Config.LDAP.Enable {
			ldapUser, err := ldap.Auth(username, password)
			if err == nil {
				// LDAP Auth Success
				// Sync to Local DB (JIT)
				var user userModels.InsightUsers
				tx := global.App.DB.Where("username = ?", username).First(&user)
				if tx.RowsAffected == 0 {
					// Create new user
					user = userModels.InsightUsers{
						Username: ldapUser.Username,
						NickName: ldapUser.Nickname,
						Email:    ldapUser.Email,
						Mobile:   ldapUser.Mobile,
						Password: "", // No local password
						IsActive: true,
					}
					global.App.DB.Create(&user)
				} else {
					// Update existing user info
					user.NickName = ldapUser.Nickname
					user.Email = ldapUser.Email
					user.Mobile = ldapUser.Mobile
					global.App.DB.Save(&user)
				}

				// Skip local password check and proceed
				c.Set("loginUserName", username)
				c.Set("loginNeedsOTP", "NO") // Assuming LDAP handles 2FA or we skip it
				c.Next()
				return
			}
			global.App.Log.Warnf("LDAP login failed for %s: %v, falling back to local", username, err)
		}

		check := userInfo{username: username, password: password}
		// 检查用户是否存在
		if err := check.checkIfUserExist(); err != nil {
			global.App.Log.WithFields(logrus.Fields{"request_id": requestid.Get(c), "username": username, "error": err}).Error(err.Error())
			c.AbortWithStatusJSON(500, gin.H{"message": err.Error(), "data": nil, "request_id": requestid.Get(c)})
			return
		}
		// 验证用户有效性
		if err := check.checkUserValid(); err != nil {
			global.App.Log.WithFields(logrus.Fields{"request_id": requestid.Get(c), "username": username, "error": err}).Error(err.Error())
			c.AbortWithStatusJSON(500, gin.H{"message": err.Error(), "data": nil, "request_id": requestid.Get(c)})
			return
		}
		// 用户是否开启2FA认证
		otpCode := loginVals.OtpCode
		needsOTP := check.checkIfUserNeedsOTP()
		if needsOTP && otpCode == "" {
			if check.checkIfOtpSecretNullString() {
				// 秘钥为空，提醒用户重新绑定otpCode
				err := errors.New("OtpSecret为空，请重新绑定OTP码")
				global.App.Log.WithFields(logrus.Fields{"request_id": requestid.Get(c), "username": username, "error": err}).Error(err.Error())
				c.AbortWithStatusJSON(401, gin.H{"status": "otp_rebind", "message": err.Error()})
				return
			}
			err := errors.New("OTP required")
			global.App.Log.WithFields(logrus.Fields{"request_id": requestid.Get(c), "username": username, "error": err}).Error(err.Error())
			c.AbortWithStatusJSON(401, gin.H{"status": "otp_required", "message": err.Error()})
			return
		}
		c.Set("loginUserName", username)
		c.Set("loginOtpCode", loginVals.OtpCode)
		if needsOTP {
			c.Set("loginNeedsOTP", "YES")
		} else {
			c.Set("loginNeedsOTP", "NO")
		}

		c.Next()
	}
}

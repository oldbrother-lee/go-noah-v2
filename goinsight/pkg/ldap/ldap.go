package ldap

import (
	"crypto/tls"
	"fmt"
	"goInsight/global"

	"github.com/go-ldap/ldap/v3"
)

type UserInfo struct {
	Username string
	Nickname string
	Email    string
	Mobile   string
}

func Auth(username, password string) (*UserInfo, error) {
	cfg := global.App.Config.LDAP
	if !cfg.Enable {
		return nil, fmt.Errorf("LDAP not enabled")
	}

	// 1. Connect to LDAP
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	var conn *ldap.Conn
	var err error
	if cfg.UseSSL {
		conn, err = ldap.DialTLS("tcp", addr, &tls.Config{InsecureSkipVerify: true})
	} else {
		conn, err = ldap.Dial("tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("LDAP connect failed: %v", err)
	}
	defer conn.Close()

	// 2. Bind with Admin User (to search)
	if cfg.BindDN != "" && cfg.BindPass != "" {
		err = conn.Bind(cfg.BindDN, cfg.BindPass)
		if err != nil {
			return nil, fmt.Errorf("LDAP admin bind failed: %v", err)
		}
	}

	// 3. Search for the user
	filter := fmt.Sprintf(cfg.UserFilter, username)
	searchRequest := ldap.NewSearchRequest(
		cfg.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		nil, // Fetch all attributes for debugging
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil || len(sr.Entries) != 1 {
		return nil, fmt.Errorf("LDAP user not found or multiple entries: %v", err)
	}

	userEntry := sr.Entries[0]

	// Debug: Print all attributes to help identify the correct nickname field
	// fmt.Printf("--- LDAP Debug Info for user: %s ---\n", username)
	// for _, attr := range userEntry.Attributes {
	// 	fmt.Printf("Attribute: %s, Values: %v\n", attr.Name, attr.Values)
	// }
	// fmt.Println("----------------------------------------")

	userDN := userEntry.DN

	// 4. Bind with the found User (Verify Password)
	// Create new connection for user bind to avoid messing up admin session if needed,
	// but usually re-bind on same conn is fine or safer to new conn.
	// For simplicity, we re-bind here.
	err = conn.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("LDAP user bind failed (password mismatch): %v", err)
	}

	// 5. Return User Info
	return &UserInfo{
		Username: username,
		Nickname: userEntry.GetAttributeValue(cfg.Attributes.Nickname),
		Email:    userEntry.GetAttributeValue(cfg.Attributes.Email),
		Mobile:   userEntry.GetAttributeValue(cfg.Attributes.Mobile),
	}, nil
}

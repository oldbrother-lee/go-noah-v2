declare namespace Api {
  /**
   * namespace Admin
   *
   * backend api module: "admin"
   */
  namespace Admin {
    // 登录相关
    interface LoginRequest {
      username: string;
      password: string;
    }

    interface LoginResponse {
      accessToken: string;
    }

    // 菜单相关
    interface Menu {
      id: number;
      parentId?: number;
      weight: number;
      path: string;
      title: string;
      name?: string;
      component?: string;
      locale?: string;
      icon?: string;
      redirect?: string;
      keepAlive?: boolean;
      hideInMenu?: boolean;
      url?: string;
      updatedAt?: string;
    }

    interface MenuListResponse {
      list: Menu[];
    }

    interface MenuCreateRequest {
      parentId?: number;
      weight: number;
      path: string;
      title: string;
      name?: string;
      component?: string;
      locale?: string;
      icon?: string;
      redirect?: string;
      keepAlive?: boolean;
      hideInMenu?: boolean;
      url?: string;
    }

    interface MenuUpdateRequest extends MenuCreateRequest {
      id: number;
    }

    // 用户相关
    interface AdminUser {
      id: number;
      username: string;
      nickname: string;
      email: string;
      phone: string;
      roles: string[];
      createdAt: string;
      updatedAt: string;
    }

    interface AdminUserListResponse {
      list: AdminUser[];
      total: number;
    }

    interface AdminUserCreateRequest {
      username: string;
      nickname: string;
      password: string;
      email: string;
      phone: string;
      roles: string[];
    }

    interface AdminUserUpdateRequest extends AdminUserCreateRequest {
      id: number;
      password?: string; // 可选，不传则不更新密码
    }

    // 角色相关
    interface Role {
      id: number;
      name: string;
      sid: string;
      createdAt: string;
      updatedAt: string;
    }

    interface RoleListResponse {
      list: Role[];
      total: number;
    }

    interface RoleCreateRequest {
      sid: string;
      name: string;
    }

    interface RoleUpdateRequest extends RoleCreateRequest {
      id: number;
    }

    // API相关
    interface Api {
      id: number;
      group: string;
      name: string;
      path: string;
      method: string;
      createdAt: string;
      updatedAt: string;
    }

    interface ApiListResponse {
      list: Api[];
      total: number;
      groups: string[];
    }

    interface ApiCreateRequest {
      group: string;
      name: string;
      path: string;
      method: string;
    }

    interface ApiUpdateRequest extends ApiCreateRequest {
      id: number;
    }

    // 权限相关
    interface Permission {
      type: 'menu' | 'api';
      resource: string;
      action: string;
    }

    interface UserPermissionsResponse {
      list: string[]; // 格式: "menu:/path" 或 "api:/path,GET"
    }

    interface RolePermissionsRequest {
      role: string;
    }

    interface RolePermissionsResponse {
      list: string[]; // 格式: "menu:/path" 或 "api:/path,GET"
    }

    interface UpdateRolePermissionRequest {
      role: string;
      list: string[]; // 格式: "menu:/path" 或 "api:/path,GET"
    }
  }
}


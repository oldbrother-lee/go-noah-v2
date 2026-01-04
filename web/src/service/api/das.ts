import { request } from '../request';

/** DAS (Data Access Service) API */

/**
 * Get environments
 */
export function fetchEnvironments(params?: Record<string, any>) {
  return request<Api.Das.Environment[]>({
    url: '/api/v1/das/environments',
    method: 'get',
    params
  });
}

/**
 * Get authorized schemas
 */
export function fetchSchemas(params?: Record<string, any>) {
  return request<Api.Das.Schema[]>({
    url: '/api/v1/das/schemas',
    method: 'get',
    params
  });
}

/**
 * Get authorized tables
 */
export function fetchTables(params?: Record<string, any>) {
  return request<Api.Das.Table[]>({
    url: '/api/v1/das/tables',
    method: 'get',
    params
  });
}

/**
 * Execute MySQL/TiDB query
 */
export function fetchExecuteMySQLQuery(data: Api.Das.QueryRequest) {
  return request<Api.Das.QueryResult>({
    url: '/api/v1/das/execute/query/mysql',
    method: 'post',
    data,
    skipErrorHandler: true
  } as any);
}

/**
 * Execute ClickHouse query
 */
export function fetchExecuteClickHouseQuery(data: Api.Das.QueryRequest) {
  return request<Api.Das.QueryResult>({
    url: '/api/v1/das/execute/query/clickhouse',
    method: 'post',
    data,
    skipErrorHandler: true
  } as any);
}

/**
 * Get user grants
 */
export function fetchUserGrants(params?: Record<string, any>) {
  return request<Api.Das.UserGrant>({
    url: '/api/v1/das/user/grants',
    method: 'get',
    params
  });
}

/**
 * Get database dictionary
 */
export function fetchDBDict(params?: Record<string, any>) {
  return request<Api.Das.DBDict>({
    url: '/api/v1/das/dbdict',
    method: 'get',
    params
  });
}

/**
 * Get query history
 */
export function fetchHistory(params?: Record<string, any>) {
  return request<Api.Das.History[]>({
    url: '/api/v1/das/history',
    method: 'get',
    params
  });
}

/**
 * Get favorites
 */
export function fetchFavorites(params?: Record<string, any>) {
  return request<Api.Das.Favorite[]>({
    url: '/api/v1/das/favorites',
    method: 'get',
    params
  });
}

/**
 * Create favorite
 */
export function fetchCreateFavorite(data: Api.Das.CreateFavoriteRequest) {
  return request<Api.Das.Favorite>({
    url: '/api/v1/das/favorites',
    method: 'post',
    data
  });
}

/**
 * Update favorite
 */
export function fetchUpdateFavorite(data: Api.Das.UpdateFavoriteRequest) {
  return request<Api.Das.Favorite>({
    url: `/api/v1/das/favorites/${data.id}`,
    method: 'put',
    data
  });
}

/**
 * Delete favorite
 */
export function fetchDeleteFavorite(id: string | number) {
  return request({
    url: `/api/v1/das/favorites/${id}`,
    method: 'delete'
  });
}

/**
 * Get table info
 */
export function fetchTableInfo(params?: Record<string, any>) {
  return request<Api.Das.TableInfo>({
    url: '/api/v1/das/table-info',
    method: 'get',
    params
  });
}

// Admin APIs for DAS management

/**
 * Admin: Get schemas list grant
 */
export function fetchAdminSchemasListGrant(params?: Record<string, any>) {
  return request<Api.Das.SchemaGrant[]>({
    url: '/api/v1/das/admin/schemas/grant',
    method: 'get',
    params
  });
}

/**
 * Admin: Create schemas grant
 */
export function fetchAdminCreateSchemasGrant(data: Api.Das.CreateSchemaGrantRequest) {
  return request({
    url: '/api/v1/das/admin/schemas/grant',
    method: 'post',
    data
  });
}

/**
 * Admin: Delete schemas grant
 */
export function fetchAdminDeleteSchemasGrant(id: string | number) {
  return request({
    url: `/api/v1/das/admin/schemas/grant/${id}`,
    method: 'delete'
  });
}

/**
 * Admin: Get tables grant
 */
export function fetchAdminTablesGrant(params?: Record<string, any>) {
  return request<Api.Das.TableGrant[]>({
    url: '/api/v1/das/admin/tables/grant',
    method: 'get',
    params
  });
}

/**
 * Admin: Create tables grant
 */
export function fetchAdminCreateTablesGrant(data: Api.Das.CreateTableGrantRequest) {
  return request({
    url: '/api/v1/das/admin/tables/grant',
    method: 'post',
    data
  });
}

/**
 * Admin: Delete tables grant
 */
export function fetchAdminDeleteTablesGrant(id: string | number) {
  return request({
    url: `/api/v1/das/admin/tables/grant/${id}`,
    method: 'delete'
  });
}

/**
 * Admin: Get instances list
 */
export function fetchAdminInstancesList(params?: Record<string, any>) {
  return request<Api.Das.Instance[]>({
    url: '/api/v1/das/admin/instances/list',
    method: 'get',
    params
  });
}

/**
 * Admin: Get schemas list
 */
export function fetchAdminSchemasList(params?: Record<string, any>) {
  return request<Api.Das.Schema[]>({
    url: '/api/v1/das/admin/schemas/list',
    method: 'get',
    params
  });
}

/**
 * Admin: Get tables list
 */
export function fetchAdminTablesList(params?: Record<string, any>) {
  return request<Api.Das.Table[]>({
    url: '/api/v1/das/admin/tables/list',
    method: 'get',
    params
  });
}

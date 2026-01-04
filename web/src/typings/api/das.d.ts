declare namespace Api {
  /**
   * namespace Das
   *
   * backend api module: "das" (Data Access Service)
   */
  namespace Das {
    interface Environment {
      id: number;
      name: string;
      description?: string;
    }

    interface Schema {
      id: number;
      name: string;
      instanceId: number;
      instanceName: string;
    }

    interface Table {
      id: number;
      name: string;
      schemaId: number;
      schemaName: string;
      comment?: string;
    }

    interface QueryRequest {
      sql: string;
      instanceId: number;
      schemaName: string;
      limit?: number;
    }

    interface QueryResult {
      columns: string[];
      rows: any[][];
      affectedRows?: number;
      executionTime: number;
      queryId?: string;
    }

    interface UserGrant {
      id: number;
      userId: number;
      resourceType: string;
      resourceId: number;
      permissions: string[];
    }

    interface DBDict {
      tables: TableInfo[];
    }

    interface TableInfo {
      tableName: string;
      tableComment?: string;
      columns: ColumnInfo[];
      indexes: IndexInfo[];
    }

    interface ColumnInfo {
      columnName: string;
      dataType: string;
      isNullable: boolean;
      columnDefault?: string;
      columnComment?: string;
      isPrimaryKey: boolean;
    }

    interface IndexInfo {
      indexName: string;
      columnNames: string[];
      isUnique: boolean;
    }

    interface History {
      id: number;
      sql: string;
      instanceName: string;
      schemaName: string;
      executionTime: string;
      duration: number;
      status: string;
    }

    interface Favorite {
      id: number;
      created_at: string;
      updated_at: string;
      username: string;
      title: string;
      sqltext: string;
    }

    interface CreateFavoriteRequest {
      title: string;
      sqltext: string;
    }

    interface UpdateFavoriteRequest {
      id: number;
      title: string;
      sqltext: string;
    }

    interface SchemaGrant {
      id: number;
      userId: number;
      userName: string;
      schemaId: number;
      schemaName: string;
      permissions: string[];
    }

    interface CreateSchemaGrantRequest {
      userId: number;
      schemaId: number;
      permissions: string[];
    }

    interface TableGrant {
      id: number;
      userId: number;
      userName: string;
      tableId: number;
      tableName: string;
      permissions: string[];
    }

    interface CreateTableGrantRequest {
      userId: number;
      tableId: number;
      permissions: string[];
    }

    interface Instance {
      id: number;
      name: string;
      host: string;
      port: number;
      dbType: string;
      environment: string;
    }
  }
}

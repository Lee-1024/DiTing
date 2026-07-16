export interface OperationLog {
  id: string;
  userId: string;
  username: string;
  method: string;
  path: string;
  status: number;
  ip: string;
  userAgent: string;
  createdAt: string;
}

export interface OperationLogQuery {
  start_time?: string;
  end_time?: string;
  username?: string;
  method?: string;
  keyword?: string;
  status?: number;
  page?: number;
  page_size?: number;
}

export interface PagedOperationLogs {
  items: OperationLog[];
  page: number;
  pageSize: number;
  total: number;
}

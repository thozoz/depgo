export interface Resp<T = unknown> {
  code: number;
  message: string;
  data: T;
}

export interface User {
  id: number;
  username: string;
  role?: number[];
  role_names?: string[];
}

export interface LoginRespData {
  token: string;
  device_key?: string;
}

export interface FileItem {
  id: string;
  path: string;
  virtual_path?: string;
  name: string;
  size: number;
  is_dir: boolean;
  modified: string;
  created?: string;
  sign?: string;
  thumb?: string;
  type: number;
}

export interface FsListResp {
  content: FileItem[] | null;
  total: number;
  filtered_total?: number;
  page: number;
  per_page: number;
  has_more?: boolean;
  write?: boolean;
}

export interface FsGetResp extends FileItem {
  raw_url: string;
  readme?: string;
  provider?: string;
  web_proxy?: boolean;
}

export interface ShareResp {
  id: number;
  share_id: string;
  name: string;
  root_path: string;
  is_dir: boolean;
  has_password: boolean;
  burn_after_read: boolean;
  access_limit: number;
  access_count: number;
  remaining_accesses: number;
  allow_preview: boolean;
  allow_download: boolean;
  enabled: boolean;
  view_count: number;
  download_count: number;
  last_access_at?: string | null;
  consumed_at?: string | null;
  expires_at?: string | null;
  created_at: string;
  updated_at: string;
  url?: string;
}

export interface ShareListResp {
  content: ShareResp[];
  total: number;
}

export interface CreateShareReq {
  path: string;
  share_id?: string;
  name?: string;
  password?: string;
  expire_at?: string;
  expire_hours?: number;
  access_limit?: number;
  burn_after_read?: boolean;
  allow_preview?: boolean;
  allow_download?: boolean;
}

export interface UpdateShareReq {
  share_id: string;
  new_share_id?: string;
  name?: string;
  password?: string;
  remove_password?: boolean;
  burn_after_read?: boolean;
  expire_at?: string | null;
  access_limit?: number;
  allow_preview?: boolean;
  allow_download?: boolean;
}

export interface PublicShareInfoResp {
  share_id: string;
  name: string;
  is_dir: boolean;
  has_password: boolean;
  burn_after_read: boolean;
  access_limit: number;
  access_count: number;
  remaining_accesses: number;
  allow_preview: boolean;
  allow_download: boolean;
  authed: boolean;
  consumed_at?: string | null;
  expires_at?: string | null;
  created_at: string;
}

export interface PublicShareObjResp {
  name: string;
  size: number;
  is_dir: boolean;
  modified: string;
  created?: string;
  thumb?: string;
  type: number;
  path: string;
  download_url?: string;
  preview_url?: string;
}

export interface PublicShareListResp {
  content: PublicShareObjResp[];
  total: number;
  page: number;
  per_page: number;
  has_more: boolean;
  pages_total?: number;
}

export interface PublicShareGetResp {
  item: PublicShareObjResp;
  provider: string;
}

export interface TrashItem {
  trash_id: string | number;
  original_path: string;
  size: number;
  is_dir: boolean;
  deleted_at: string;
}

export interface TrashListResp {
  content: TrashItem[];
  total: number;
  page: number;
  per_page: number;
}

export interface RestoreTrashReq {
  path?: string;
  trash_id: string | number;
  target_path?: string;
}

export interface DeleteTrashReq {
  path?: string;
  trash_id: string | number;
  confirm: true;
}

export interface FileVersionItem {
  version_id: string | number;
  revision: number;
  size: number;
  created_at: string;
}

export interface VersionsResp {
  file_id: string | number;
  revision: number;
  content: FileVersionItem[];
  total: number;
  page: number;
  per_page: number;
}

export interface RestoreVersionReq {
  path: string;
  version_id: string | number;
  revision: number;
}

export interface TextGetResp {
  content: string;
  revision: number;
}

export interface TextPutReq {
  path: string;
  content: string;
  revision: number;
}

export interface TextPutResp {
  revision: number;
}


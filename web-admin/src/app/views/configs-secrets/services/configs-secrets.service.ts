import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface EnvGroup {
  id?: string;
  code: string;
  name: string;
  description?: string;
  type: 'config' | 'secret';
  scope: 'global' | 'tenant';
  tenantID?: string;
  vnamespace?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface EnvVar {
  id?: string;
  groupID?: string;
  key: string;
  value: string;
  description?: string;
  createdAt?: string;
  updatedAt?: string;
  showSecret?: boolean; // UI local state for toggling eye icon
}

@Injectable({
  providedIn: 'root'
})
export class ConfigsSecretsService {
  private globalUrl = '/rest-api/env-groups';
  private tenantUrl = '/rest-api/tenants';

  constructor(private http: HttpClient) {}

  // --- GLOBAL API ---
  getGlobalGroups(pageSize: number = 50, cursor: string = ''): Observable<any> {
    return this.http.get(`${this.globalUrl}?pageSize=${pageSize}&cursor=${cursor}`);
  }

  getGlobalGroup(id: string): Observable<EnvGroup> {
    return this.http.get<EnvGroup>(`${this.globalUrl}/${id}`);
  }

  createGlobalGroup(group: Partial<EnvGroup>): Observable<EnvGroup> {
    return this.http.post<EnvGroup>(this.globalUrl, group);
  }

  updateGlobalGroup(id: string, group: Partial<EnvGroup>): Observable<EnvGroup> {
    return this.http.put<EnvGroup>(`${this.globalUrl}/${id}`, group);
  }

  deleteGlobalGroup(id: string): Observable<any> {
    return this.http.delete(`${this.globalUrl}/${id}`);
  }

  getGlobalVars(groupId: string): Observable<any> {
    return this.http.get(`${this.globalUrl}/${groupId}/vars`);
  }

  saveGlobalVar(groupId: string, envVar: Partial<EnvVar>): Observable<EnvVar> {
    return this.http.post<EnvVar>(`${this.globalUrl}/${groupId}/vars`, envVar);
  }

  deleteGlobalVar(groupId: string, varId: string): Observable<any> {
    return this.http.delete(`${this.globalUrl}/${groupId}/vars/${varId}`);
  }

  bulkSaveGlobalVars(groupId: string, vars: EnvVar[]): Observable<any> {
    return this.http.put(`${this.globalUrl}/${groupId}/vars/bulk`, { vars });
  }

  // --- TENANT API ---
  getTenantGroups(tenantCode: string, pageSize: number = 50, cursor: string = ''): Observable<any> {
    return this.http.get(`${this.tenantUrl}/${tenantCode}/env-groups?pageSize=${pageSize}&cursor=${cursor}`);
  }

  getTenantGroup(tenantCode: string, id: string): Observable<EnvGroup> {
    return this.http.get<EnvGroup>(`${this.tenantUrl}/${tenantCode}/env-groups/${id}`);
  }

  createTenantGroup(tenantCode: string, group: Partial<EnvGroup>): Observable<EnvGroup> {
    return this.http.post<EnvGroup>(`${this.tenantUrl}/${tenantCode}/env-groups`, group);
  }

  updateTenantGroup(tenantCode: string, id: string, group: Partial<EnvGroup>): Observable<EnvGroup> {
    return this.http.put<EnvGroup>(`${this.tenantUrl}/${tenantCode}/env-groups/${id}`, group);
  }

  deleteTenantGroup(tenantCode: string, id: string): Observable<any> {
    return this.http.delete(`${this.tenantUrl}/${tenantCode}/env-groups/${id}`);
  }

  getTenantVars(tenantCode: string, groupId: string): Observable<any> {
    return this.http.get(`${this.tenantUrl}/${tenantCode}/env-groups/${groupId}/vars`);
  }

  saveTenantVar(tenantCode: string, groupId: string, envVar: Partial<EnvVar>): Observable<EnvVar> {
    return this.http.post<EnvVar>(`${this.tenantUrl}/${tenantCode}/env-groups/${groupId}/vars`, envVar);
  }

  deleteTenantVar(tenantCode: string, groupId: string, varId: string): Observable<any> {
    return this.http.delete(`${this.tenantUrl}/${tenantCode}/env-groups/${groupId}/vars/${varId}`);
  }

  bulkSaveTenantVars(tenantCode: string, groupId: string, vars: EnvVar[]): Observable<any> {
    return this.http.put(`${this.tenantUrl}/${tenantCode}/env-groups/${groupId}/vars/bulk`, { vars });
  }
}

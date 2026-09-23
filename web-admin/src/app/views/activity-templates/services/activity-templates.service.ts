import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface ActivityTemplate {
  id?: string;
  code: string;
  vnamespace?: string;
  name: string;
  description?: string;
  activityFamily: string;
  parentTemplateId: string;
  rootActivity?: string;
  payload?: string;
  isActive: boolean;
  scope: 'global' | 'tenant';
  tenantId?: string;
  createdAt?: string;
  updatedAt?: string;
}

@Injectable({
  providedIn: 'root'
})
export class ActivityTemplatesService {
  private globalUrl = '/rest-api/activity-templates';
  private tenantUrl = '/rest-api/tenants';

  constructor(private http: HttpClient) {}

  // --- GLOBAL API ---
  getGlobalTemplates(
    pageSize: number = 50,
    cursor: string = '',
    vnamespace: string = '',
    activityFamily: string = ''
  ): Observable<any> {
    let params = `pageSize=${pageSize}&cursor=${encodeURIComponent(cursor)}`;
    if (vnamespace) {
      params += `&vnamespace=${encodeURIComponent(vnamespace)}`;
    }
    if (activityFamily) {
      params += `&activityFamily=${encodeURIComponent(activityFamily)}`;
    }
    return this.http.get(`${this.globalUrl}?${params}`);
  }

  getGlobalTemplate(id: string): Observable<ActivityTemplate> {
    return this.http.get<ActivityTemplate>(`${this.globalUrl}/${id}`);
  }

  createGlobalTemplate(tpl: Partial<ActivityTemplate>): Observable<ActivityTemplate> {
    return this.http.post<ActivityTemplate>(this.globalUrl, tpl);
  }

  updateGlobalTemplate(id: string, tpl: Partial<ActivityTemplate>): Observable<ActivityTemplate> {
    return this.http.put<ActivityTemplate>(`${this.globalUrl}/${id}`, tpl);
  }

  deleteGlobalTemplate(id: string): Observable<any> {
    return this.http.delete(`${this.globalUrl}/${id}`);
  }

  // --- TENANT API ---
  getTenantTemplates(
    tenantCode: string,
    pageSize: number = 50,
    cursor: string = '',
    vnamespace: string = '',
    activityFamily: string = ''
  ): Observable<any> {
    let params = `pageSize=${pageSize}&cursor=${encodeURIComponent(cursor)}`;
    if (vnamespace) {
      params += `&vnamespace=${encodeURIComponent(vnamespace)}`;
    }
    if (activityFamily) {
      params += `&activityFamily=${encodeURIComponent(activityFamily)}`;
    }
    return this.http.get(`${this.tenantUrl}/${tenantCode}/activity-templates?${params}`);
  }

  getTenantTemplate(tenantCode: string, id: string): Observable<ActivityTemplate> {
    return this.http.get<ActivityTemplate>(`${this.tenantUrl}/${tenantCode}/activity-templates/${id}`);
  }

  createTenantTemplate(tenantCode: string, tpl: Partial<ActivityTemplate>): Observable<ActivityTemplate> {
    return this.http.post<ActivityTemplate>(`${this.tenantUrl}/${tenantCode}/activity-templates`, tpl);
  }

  updateTenantTemplate(tenantCode: string, id: string, tpl: Partial<ActivityTemplate>): Observable<ActivityTemplate> {
    return this.http.put<ActivityTemplate>(`${this.tenantUrl}/${tenantCode}/activity-templates/${id}`, tpl);
  }

  deleteTenantTemplate(tenantCode: string, id: string): Observable<any> {
    return this.http.delete(`${this.tenantUrl}/${tenantCode}/activity-templates/${id}`);
  }

  // --- DESIGNER LAZY FETCH (Global + Tenant if in tenant context) ---
  getForDesigner(tenantCode: string = '', pageSize: number = 20, cursor: string = ''): Observable<any> {
    let params = `pageSize=${pageSize}&cursor=${encodeURIComponent(cursor)}`;
    if (tenantCode) {
      params += `&tenantCode=${encodeURIComponent(tenantCode)}`;
    }
    return this.http.get(`${this.globalUrl}/for-designer?${params}`);
  }

  decodePayload(payloadRaw: any): string {
    if (!payloadRaw) return '';
    if (typeof payloadRaw === 'string') {
      const trimmed = payloadRaw.trim();
      if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
        return trimmed;
      }
      try {
        return atob(trimmed);
      } catch {
        return trimmed;
      }
    }
    return JSON.stringify(payloadRaw, null, 2);
  }
}

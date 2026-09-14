import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface WorkflowDefinition {
  id?: string;
  code: string;
  vnamespace?: string;
  name: string;
  description?: string;
  version: number;
  payload?: string;
  payloadFormat: 'json' | 'yaml' | 'bpmn';
  maxDurationSeconds: number;
  isActive: boolean;
  scope: 'global' | 'tenant';
  tenantId?: string;
  createdAt?: string;
  updatedAt?: string;
}

@Injectable({
  providedIn: 'root'
})
export class WorkflowsService {
  private globalUrl = '/rest-api/workflows';
  private tenantUrl = '/rest-api/tenants';

  constructor(private http: HttpClient) {}

  // --- GLOBAL API ---
  getGlobalWorkflows(pageSize: number = 50, cursor: string = '', vnamespace: string = ''): Observable<any> {
    let params = `pageSize=${pageSize}&cursor=${cursor}`;
    if (vnamespace) {
      params += `&vnamespace=${encodeURIComponent(vnamespace)}`;
    }
    return this.http.get(`${this.globalUrl}?${params}`);
  }

  getGlobalWorkflow(id: string): Observable<WorkflowDefinition> {
    return this.http.get<WorkflowDefinition>(`${this.globalUrl}/${id}`);
  }

  createGlobalWorkflow(wf: Partial<WorkflowDefinition>): Observable<WorkflowDefinition> {
    return this.http.post<WorkflowDefinition>(this.globalUrl, wf);
  }

  updateGlobalWorkflow(id: string, wf: Partial<WorkflowDefinition>): Observable<WorkflowDefinition> {
    return this.http.put<WorkflowDefinition>(`${this.globalUrl}/${id}`, wf);
  }

  deleteGlobalWorkflow(id: string): Observable<any> {
    return this.http.delete(`${this.globalUrl}/${id}`);
  }

  getGlobalWorkflowQueues(id: string): Observable<any> {
    return this.http.get(`${this.globalUrl}/${id}/queues`);
  }

  // --- TENANT API ---
  getTenantWorkflows(tenantCode: string, pageSize: number = 50, cursor: string = '', vnamespace: string = ''): Observable<any> {
    let params = `pageSize=${pageSize}&cursor=${cursor}`;
    if (vnamespace) {
      params += `&vnamespace=${encodeURIComponent(vnamespace)}`;
    }
    return this.http.get(`${this.tenantUrl}/${tenantCode}/workflows?${params}`);
  }

  getTenantWorkflow(tenantCode: string, id: string): Observable<WorkflowDefinition> {
    return this.http.get<WorkflowDefinition>(`${this.tenantUrl}/${tenantCode}/workflows/${id}`);
  }

  createTenantWorkflow(tenantCode: string, wf: Partial<WorkflowDefinition>): Observable<WorkflowDefinition> {
    return this.http.post<WorkflowDefinition>(`${this.tenantUrl}/${tenantCode}/workflows`, wf);
  }

  updateTenantWorkflow(tenantCode: string, id: string, wf: Partial<WorkflowDefinition>): Observable<WorkflowDefinition> {
    return this.http.put<WorkflowDefinition>(`${this.tenantUrl}/${tenantCode}/workflows/${id}`, wf);
  }

  deleteTenantWorkflow(tenantCode: string, id: string): Observable<any> {
    return this.http.delete(`${this.tenantUrl}/${tenantCode}/workflows/${id}`);
  }

  getTenantWorkflowQueues(tenantCode: string, id: string): Observable<any> {
    return this.http.get(`${this.tenantUrl}/${tenantCode}/workflows/${id}/queues`);
  }
}

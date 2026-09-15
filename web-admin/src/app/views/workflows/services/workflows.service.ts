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

export interface WorkflowExecution {
  id: string;
  workflowDefinitionId: string;
  workflowDefinitionVersion: number;
  vnamespace?: string;
  executionKey?: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'terminated' | 'cancelled';
  input?: any;
  output?: any;
  stateData?: any;
  error?: string;
  startedAt?: string;
  completedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ExecutionToken {
  id: string;
  workflowExecutionId: string;
  workflowDefinitionId: string;
  vnamespace?: string;
  currentNodeId: string;
  status: string;
  parentTokenId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface WorkflowJob {
  id: string;
  workflowExecutionId: string;
  executionTokenId?: string;
  workflowDefinitionId?: string;
  vnamespace?: string;
  activityId?: string;
  activityName?: string;
  activityType?: string;
  status: string;
  input?: any;
  output?: any;
  error?: string;
  assignedWorkerId?: string;
  retries?: number;
  maxRetries?: number;
  timeoutSeconds?: number;
  createdAt?: string;
  updatedAt?: string;
}

export interface WorkflowExecutionDetail {
  execution: WorkflowExecution;
  tokens: ExecutionToken[];
  jobs: WorkflowJob[];
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

  // --- EXECUTION API ---
  executeGlobalWorkflow(workflowDefinitionId: string, input: any = {}, executionKey: string = '', vnamespace: string = ''): Observable<any> {
    return this.http.post(`${this.globalUrl}/executions`, {
      workflowDefinitionId,
      executionKey,
      input,
      vnamespace
    });
  }

  executeTenantWorkflow(tenantCode: string, workflowDefinitionId: string, input: any = {}, executionKey: string = '', vnamespace: string = ''): Observable<any> {
    return this.http.post(`${this.tenantUrl}/${tenantCode}/workflow-executions`, {
      workflowDefinitionId,
      executionKey,
      input,
      vnamespace
    });
  }

  getGlobalExecutions(workflowDefinitionId?: string, status?: string, pageSize: number = 50, cursor: string = '', vnamespace: string = ''): Observable<any> {
    let params = `pageSize=${pageSize}&cursor=${cursor}`;
    if (workflowDefinitionId) {
      params += `&workflowDefinitionId=${encodeURIComponent(workflowDefinitionId)}`;
    }
    if (status) {
      params += `&status=${encodeURIComponent(status)}`;
    }
    if (vnamespace) {
      params += `&vnamespace=${encodeURIComponent(vnamespace)}`;
    }
    return this.http.get(`${this.globalUrl}/executions?${params}`);
  }

  getTenantExecutions(tenantCode: string, workflowDefinitionId?: string, status?: string, pageSize: number = 50, cursor: string = '', vnamespace: string = ''): Observable<any> {
    let params = `pageSize=${pageSize}&cursor=${cursor}`;
    if (workflowDefinitionId) {
      params += `&workflowDefinitionId=${encodeURIComponent(workflowDefinitionId)}`;
    }
    if (status) {
      params += `&status=${encodeURIComponent(status)}`;
    }
    if (vnamespace) {
      params += `&vnamespace=${encodeURIComponent(vnamespace)}`;
    }
    return this.http.get(`${this.tenantUrl}/${tenantCode}/workflow-executions?${params}`);
  }

  getGlobalExecutionDetail(id: string): Observable<WorkflowExecutionDetail> {
    return this.http.get<WorkflowExecutionDetail>(`${this.globalUrl}/executions/${id}`);
  }

  getTenantExecutionDetail(tenantCode: string, id: string): Observable<WorkflowExecutionDetail> {
    return this.http.get<WorkflowExecutionDetail>(`${this.tenantUrl}/${tenantCode}/workflow-executions/${id}`);
  }
}

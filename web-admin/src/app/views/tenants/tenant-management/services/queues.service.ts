import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class QueuesService {
  private apiUrl = '/rest-api/tenants';

  constructor(private http: HttpClient) { }

  getQueues(tenantId: string, cursor: string = '', pageSize: number = 10, q: string = '', vnamespace: string = ''): Observable<any> {
    let params = `cursor=${cursor}&pageSize=${pageSize}&q=${q}`;
    if (vnamespace) {
      params += `&vnamespace=${vnamespace}`;
    }
    return this.http.get(`${this.apiUrl}/${tenantId}/queue?${params}`);
  }

  getQueue(tenantId: string, queueId: string): Observable<any> {
    return this.http.get(`${this.apiUrl}/${tenantId}/queue/${queueId}`);
  }

  createQueue(tenantId: string, queue: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/${tenantId}/queue`, queue);
  }

  bulkCreateQueues(tenantId: string, queues: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/${tenantId}/queue/bulk`, queues);
  }

  deleteQueue(tenantId: string, queueId: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${tenantId}/queue/${queueId}`);
  }
}

import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class QueuesService {
  private apiUrl = '/rest-api/tenants';

  constructor(private http: HttpClient) { }

  getQueues(tenantCode: string, cursor: string = '', pageSize: number = 10, q: string = '', vnamespace: string = '', includeHeaders: boolean = false): Observable<any> {
    let params = `cursor=${cursor}&pageSize=${pageSize}&q=${q}`;
    if (vnamespace) {
      params += `&vnamespace=${vnamespace}`;
    }
    if (includeHeaders) {
      params += `&includeHeaders=true`;
    }
    return this.http.get(`${this.apiUrl}/${tenantCode}/queue?${params}`);
  }

  getQueue(tenantCode: string, code: string, vnamespace: string): Observable<any> {
    return this.http.get(`${this.apiUrl}/${tenantCode}/queue/${code}/${vnamespace}`);
  }

  createQueue(tenantCode: string, queue: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/${tenantCode}/queue`, queue);
  }

  bulkCreateQueues(tenantCode: string, queues: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/${tenantCode}/queue/bulk`, queues);
  }

  deleteQueue(tenantCode: string, code: string, vnamespace: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${tenantCode}/queue/${code}/${vnamespace}`);
  }

  enqueueMessage(tenantCode: string, messageData: any): Observable<any> {
    const { queueCode, vnamespace, ...payload } = messageData;
    return this.http.post(`${this.apiUrl}/${tenantCode}/queue/${queueCode}/${vnamespace}/enqueue`, payload);
  }
}

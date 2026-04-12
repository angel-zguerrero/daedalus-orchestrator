import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class QueueMessagesService {
  private apiUrl = '/rest-api/tenants';

  constructor(private http: HttpClient) {}

  getQueueMessages(
    tenantCode: string,
    queueCode: string,
    vnamespace: string,
    cursor: string = '',
    pageSize: number = 20
  ): Observable<any> {
    const params = `cursor=${encodeURIComponent(cursor)}&pageSize=${pageSize}`;
    return this.http.get(
      `${this.apiUrl}/${tenantCode}/queue/${queueCode}/${vnamespace}/messages?${params}`
    );
  }
}

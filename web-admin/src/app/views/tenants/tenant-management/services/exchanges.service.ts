import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class ExchangesService {
  private apiUrl = '/rest-api/tenants';

  constructor(private http: HttpClient) { }

  getExchanges(tenantCode: string, cursor: string = '', pageSize: number = 10, q: string = '', vnamespace: string = ''): Observable<any> {
    let params = `cursor=${cursor}&pageSize=${pageSize}&q=${q}`;
    if (vnamespace) {
      params += `&vnamespace=${vnamespace}`;
    }
    return this.http.get(`${this.apiUrl}/${tenantCode}/exchange?${params}`);
  }

  getExchange(tenantCode: string, code: string, vnamespace: string): Observable<any> {
    return this.http.get(`${this.apiUrl}/${tenantCode}/exchange/${code}/${vnamespace}`);
  }

  createExchange(tenantCode: string, exchange: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/${tenantCode}/exchange`, exchange);
  }

  bulkCreateExchanges(tenantCode: string, exchanges: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/${tenantCode}/exchange/bulk`, exchanges);
  }

  deleteExchange(tenantCode: string, code: string, vnamespace: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${tenantCode}/exchange/${code}/${vnamespace}`);
  }

  publishMessage(tenantCode: string, messageData: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/${tenantCode}/exchange/publish-message`, messageData);
  }
}

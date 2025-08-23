import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class ExchangesService {
  private apiUrl = '/rest-api/tenants';

  constructor(private http: HttpClient) { }

  getExchanges(tenantId: string, cursor: string = '', pageSize: number = 10, q: string = '', vnamespace: string = ''): Observable<any> {
    let params = `cursor=${cursor}&pageSize=${pageSize}&q=${q}`;
    if (vnamespace) {
      params += `&vnamespace=${vnamespace}`;
    }
    return this.http.get(`${this.apiUrl}/${tenantId}/exchange?${params}`);
  }

  getExchange(tenantId: string, code: string, vnamespace: string): Observable<any> {
    return this.http.get(`${this.apiUrl}/${tenantId}/exchange/${code}/${vnamespace}`);
  }

  createExchange(tenantId: string, exchange: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/${tenantId}/exchange`, exchange);
  }

  bulkCreateExchanges(tenantId: string, exchanges: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/${tenantId}/exchange/bulk`, exchanges);
  }

  deleteExchange(tenantId: string, code: string, vnamespace: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${tenantId}/exchange/${code}/${vnamespace}`);
  }
}

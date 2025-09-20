import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class BindingsService {
  private apiUrl = '/rest-api/tenants';

  constructor(private http: HttpClient) { }

  getBindings(tenantCode: string, cursor: string = '', pageSize: number = 10, q: string = '', vnamespace: string = '', includeObjects: boolean = false): Observable<any> {
    let params = `cursor=${cursor}&pageSize=${pageSize}&q=${q}&includeObjects=${includeObjects}`;
    if (vnamespace) {
      params += `&vnamespace=${vnamespace}`;
    }
    return this.http.get(`${this.apiUrl}/${tenantCode}/bindings?${params}`);
  }

  getBinding(tenantCode: string, code: string, vnamespace: string): Observable<any> {
    return this.http.get(`${this.apiUrl}/${tenantCode}/binding/${code}/${vnamespace}`);
  }

  createBinding(tenantCode: string, binding: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/${tenantCode}/binding`, binding);
  }

  deleteBinding(tenantCode: string, code: string, vnamespace: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${tenantCode}/binding/${code}/${vnamespace}`);
  }
}

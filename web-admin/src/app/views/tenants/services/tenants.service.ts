import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class TenantsService {
  private apiUrl = '/rest-api/tenants';

  constructor(private http: HttpClient) { }

  getTenants(cursor: string = '', pageSize: number = 10, q: string = ''): Observable<any> {
    return this.http.get(`${this.apiUrl}?cursor=${cursor}&pageSize=${pageSize}&q=${q}`);
  }

  getTenant(code: string): Observable<any> {
    return this.http.get(`${this.apiUrl}/${code}`);
  }

  getTenantSummary(code: string): Observable<any> {
    return this.http.get(`${this.apiUrl}/${code}/summary`);
  }

  assertTenant(tenant: any): Observable<any> {
    return this.http.post(this.apiUrl, tenant);
  }

  deleteTenant(code: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${code}`);
  }

  bulkAssertTenants(tenants: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/bulk`, tenants);
  }
}

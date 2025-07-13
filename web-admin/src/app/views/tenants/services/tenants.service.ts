import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class TenantsService {
  private apiUrl = '/admin-api/tenants';

  constructor(private http: HttpClient) { }

  getTenants(cursor: string = '', pageSize: number = 10): Observable<any> {
    return this.http.get(`${this.apiUrl}?cursor=${cursor}&pageSize=${pageSize}`);
  }

  getTenant(id: string): Observable<any> {
    return this.http.get(`${this.apiUrl}/${id}`);
  }

  createTenant(tenant: any): Observable<any> {
    return this.http.post(this.apiUrl, tenant);
  }

  updateTenant(id: string, tenant: any): Observable<any> {
    return this.http.put(`${this.apiUrl}/${id}`, tenant);
  }

  deleteTenant(id: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${id}`);
  }
}

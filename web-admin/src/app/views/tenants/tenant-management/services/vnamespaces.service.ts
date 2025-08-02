import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class VNamespacesService {
  private apiUrl = '/rest-api/tenants';

  constructor(private http: HttpClient) { }

  getVNamespaces(tenantId: string, cursor: string = '', pageSize: number = 20, q: string = ''): Observable<any> {
    return this.http.get(`${this.apiUrl}/${tenantId}/vnamespaces?cursor=${cursor}&pageSize=${pageSize}&q=${q}`);
  }
}

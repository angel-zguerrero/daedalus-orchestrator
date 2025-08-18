import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class NodeSchedulersService {
  private apiUrl = '/rest-api/node-schedulers';

  constructor(private http: HttpClient) { }

  getNodeSchedulers(cursor: string = '', pageSize: number = 10, q: string = ''): Observable<any> {
    return this.http.get(`${this.apiUrl}?cursor=${cursor}&pageSize=${pageSize}&q=${q}`);
  }

  getNodeScheduler(id: string): Observable<any> {
    return this.http.get(`${this.apiUrl}/${id}`);
  }
}

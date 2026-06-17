import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface User {
  ID: string;
  Username: string;
  Email: string;
  IsRootUser: boolean;
}

export interface FindResult<T> {
  Entities: T[];
  Cursor: string;
}

@Injectable({
  providedIn: 'root'
})
export class UsersService {
  private apiUrl = '/rest-api/users';

  constructor(private http: HttpClient) { }

  getUsers(pageSize: number = 50, cursor: string = '', query: string = ''): Observable<{ message: string, result: FindResult<User> }> {
    let params = new HttpParams()
      .set('pageSize', pageSize.toString());

    if (cursor) params = params.set('cursor', cursor);
    if (query) params = params.set('q', query);

    return this.http.get<{ message: string, result: FindResult<User> }>(this.apiUrl, { params });
  }

  createUser(user: any): Observable<any> {
    return this.http.post(this.apiUrl, user);
  }

  updateUser(id: string, user: any): Observable<any> {
    return this.http.put(`${this.apiUrl}/${id}`, user);
  }

  deleteUser(id: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${id}`);
  }
}

import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, of } from 'rxjs';
import { map, catchError } from 'rxjs/operators';

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
  private userCache: Map<string, string> = new Map<string, string>();

  constructor(private http: HttpClient) { }

  getUsers(pageSize: number = 50, cursor: string = '', query: string = ''): Observable<{ message: string, result: FindResult<User> }> {
    let params = new HttpParams()
      .set('pageSize', pageSize.toString());

    if (cursor) params = params.set('cursor', cursor);
    if (query) params = params.set('q', query);

    return this.http.get<{ message: string, result: FindResult<User> }>(this.apiUrl, { params });
  }

  getUserById(id: string): Observable<{ message: string, result: User }> {
    return this.http.get<{ message: string, result: User }>(`${this.apiUrl}/${encodeURIComponent(id)}`);
  }

  getUsersByIds(ids: string[]): Observable<{ message: string, result: User[] }> {
    return this.http.post<{ message: string, result: User[] }>(`${this.apiUrl}/by-ids`, { ids });
  }

  resolveUser(id: string): Observable<string> {
    if (!id) {
      return of('');
    }
    if (this.userCache.has(id)) {
      return of(this.userCache.get(id)!);
    }
    return this.getUserById(id).pipe(
      map(res => {
        const username = res.result?.Username || id;
        if (res.result) {
          if (res.result.ID) this.userCache.set(res.result.ID, username);
          if (res.result.Username) this.userCache.set(res.result.Username, username);
        }
        this.userCache.set(id, username);
        return username;
      }),
      catchError(() => {
        this.userCache.set(id, id);
        return of(id);
      })
    );
  }

  resolveUsers(ids: string[]): Observable<Map<string, string>> {
    const validIds = Array.from(new Set(ids.filter(id => !!id)));
    const missingIds = validIds.filter(id => !this.userCache.has(id));

    if (missingIds.length === 0) {
      return of(new Map(this.userCache));
    }

    return this.getUsersByIds(missingIds).pipe(
      map(res => {
        const users = res.result || [];
        const foundIds = new Set<string>();
        users.forEach(u => {
          if (u.ID) {
            this.userCache.set(u.ID, u.Username);
            foundIds.add(u.ID);
          }
          if (u.Username) {
            this.userCache.set(u.Username, u.Username);
            foundIds.add(u.Username);
          }
        });
        missingIds.forEach(id => {
          if (!foundIds.has(id)) {
            this.userCache.set(id, id);
          }
        });
        return new Map(this.userCache);
      }),
      catchError(() => {
        missingIds.forEach(id => this.userCache.set(id, id));
        return of(new Map(this.userCache));
      })
    );
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

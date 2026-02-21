import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
    providedIn: 'root'
})
export class JobWorkersService {
    private apiUrl = '/rest-api/job-workers';

    constructor(private http: HttpClient) { }

    getJobWorkers(cursor: string = '', pageSize: number = 10, q: string = ''): Observable<any> {
        return this.http.get(`${this.apiUrl}?cursor=${cursor}&pageSize=${pageSize}&q=${q}`);
    }

    getJobWorker(id: string): Observable<any> {
        return this.http.get(`${this.apiUrl}/${id}`);
    }
}

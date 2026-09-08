import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class ScheduledJobsService {
  private apiUrl = '/rest-api/tenants';

  constructor(private http: HttpClient) { }

  getScheduledJobs(tenantCode: string, cursor: string = '', pageSize: number = 10, vnamespace: string = ''): Observable<any> {
    let params = `cursor=${cursor}&pageSize=${pageSize}`;
    if (vnamespace) {
      params += `&vnamespace=${vnamespace}`;
    }
    return this.http.get(`${this.apiUrl}/${tenantCode}/scheduled-jobs?${params}`);
  }

  getScheduledJob(tenantCode: string, id: string): Observable<any> {
    return this.http.get(`${this.apiUrl}/${tenantCode}/scheduled-job/${id}`);
  }

  createOneOffScheduledJob(tenantCode: string, job: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/${tenantCode}/scheduled-job/one-off`, job);
  }

  createRecurringScheduledJob(tenantCode: string, job: any): Observable<any> {
    return this.http.post(`${this.apiUrl}/${tenantCode}/scheduled-job/recurring`, job);
  }

  deleteScheduledJob(tenantCode: string, id: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${tenantCode}/scheduled-job/${id}`);
  }
}

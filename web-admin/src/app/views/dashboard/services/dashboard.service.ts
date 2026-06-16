import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class DashboardService {
  private readonly apiUrl = '/rest-api/dashboard';

  constructor(private http: HttpClient) {}

  getDashboardSummary(): Observable<any> {
    return this.http.get(`${this.apiUrl}/summary`);
  }

  getGlobalTSDBMetrics(resolution: number = 5, startTime?: number, endTime?: number): Observable<any> {
    let url = `/rest-api/v1/cluster/metrics/tsdb?resolution=${resolution}`;
    if (startTime) url += `&startTime=${startTime}`;
    if (endTime) url += `&endTime=${endTime}`;
    return this.http.get(url);
  }
}

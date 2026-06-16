import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

export interface TSDBDatapoint {
  timestamp: number;
  published: number;
  delivered: number;
  acked: number;
  failed: number;
  pending: number;
  in_process: number;
  msg_per_second: number;
  avg_latency_ms: number;
  max_latency_ms: number;
}

export interface TSDBMetricsResult {
  resolution: number;
  datapoints: TSDBDatapoint[];
}

export interface TSDBMetricsResponse {
  message: string;
  result: TSDBMetricsResult;
}

@Injectable({
  providedIn: 'root'
})
export class TSDBMetricsService {
  constructor(private http: HttpClient) {}

  getTSDBMetrics(
    tenantCode: string,
    queueCode?: string,
    vnamespace?: string,
    resolution: number = 5,
    startTime?: number,
    endTime?: number
  ): Observable<TSDBMetricsResult> {
    let params = new HttpParams().set('resolution', resolution.toString());

    if (queueCode) {
      params = params.set('queueCode', queueCode);
    }
    if (vnamespace) {
      params = params.set('vnamespace', vnamespace);
    }
    if (startTime) {
      params = params.set('startTime', startTime.toString());
    }
    if (endTime) {
      params = params.set('endTime', endTime.toString());
    }

    return this.http
      .get<TSDBMetricsResponse>(`/rest-api/tenants/${tenantCode}/metrics/tsdb`, { params })
      .pipe(map(response => response.result));
  }
}

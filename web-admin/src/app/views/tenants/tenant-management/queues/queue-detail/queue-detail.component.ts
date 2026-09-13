import { Component, OnInit, Input, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  CardModule,
  AlertComponent,
  SpinnerComponent,
  BadgeComponent,
  ProgressModule,
  TooltipModule,
  GridModule
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { ChartjsModule } from '@coreui/angular-chartjs';
import { TSDBMetricsService } from '../../../services/tsdb-metrics.service';

@Component({
  selector: 'app-queue-detail',
  templateUrl: './queue-detail.component.html',
  styleUrls: [],
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    CardModule,
    AlertComponent,
    SpinnerComponent,
    BadgeComponent,
    ProgressModule,
    TooltipModule,
    GridModule,
    IconDirective,
    ChartjsModule
  ]
})
export class QueueDetailComponent implements OnInit, OnChanges {
  @Input() queue: any = null;
  @Input() tenantCode: string = '';
  @Input() isWorkflowQueue: boolean = false;

  selectedTimeRange: number = 600; // Default to last 10 minutes
  metricsLoading: boolean = false;
  metricsData: any = { labels: [], datasets: [] };
  gaugeMetricsData: any = { labels: [], datasets: [] };
  currentRates: any = { publish: 0, deliver: 0, ack: 0 };
  currentGauges: any = { pending: 0, inProcess: 0 };

  metricsOptions: any = {
    maintainAspectRatio: false,
    elements: {
      line: { tension: 0.4 },
      point: { radius: 0, hitRadius: 10, hoverRadius: 4, hoverBorderWidth: 3 }
    },
    scales: {
      y: {
        min: 0,
        ticks: {
          precision: 0,
          beginAtZero: true
        }
      }
    }
  };

  constructor(private tsdbMetricsService: TSDBMetricsService) {}

  ngOnInit(): void {
    if (this.queue) {
      this.loadMetrics();
    }
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['queue'] && this.queue) {
      this.loadMetrics();
    }
  }

  onTimeRangeChange(): void {
    this.loadMetrics();
  }

  loadMetrics(): void {
    if (!this.queue) return;
    const queueCode = this.queue.Code || this.queue.code;
    const vnamespace = this.queue.VNamespace || this.queue.vnamespace || '';
    if (!queueCode) return;

    this.metricsLoading = true;
    const endTime = Math.floor(Date.now() / 1000);
    const startTime = endTime - this.selectedTimeRange;

    const buildEmptyMetrics = () => {
      const labels: string[] = [];
      const zeroData: number[] = [];

      const normalizedStartTime = Math.floor(startTime / 5) * 5;
      const normalizedEndTime = Math.floor(endTime / 5) * 5;

      for (let ts = normalizedStartTime; ts <= normalizedEndTime; ts += 5) {
        const date = new Date(ts * 1000);
        labels.push(`${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}:${date.getSeconds().toString().padStart(2, '0')}`);
        zeroData.push(0);
      }

      this.metricsData = {
        labels: labels,
        datasets: [
          {
            label: 'Published',
            backgroundColor: 'rgba(51, 153, 255, 0.1)',
            borderColor: '#3399ff',
            pointHoverBackgroundColor: '#3399ff',
            borderWidth: 2,
            data: [...zeroData],
            fill: true
          },
          {
            label: 'Delivered',
            backgroundColor: 'rgba(249, 177, 21, 0.1)',
            borderColor: '#f9b115',
            pointHoverBackgroundColor: '#f9b115',
            borderWidth: 2,
            data: [...zeroData],
            fill: true
          },
          {
            label: 'Acked',
            backgroundColor: 'rgba(46, 184, 92, 0.1)',
            borderColor: '#2eb85c',
            pointHoverBackgroundColor: '#2eb85c',
            borderWidth: 2,
            data: [...zeroData],
            fill: true
          }
        ]
      };

      this.gaugeMetricsData = {
        labels: labels,
        datasets: [
          {
            label: 'Pending',
            backgroundColor: 'transparent',
            borderColor: 'rgba(255, 99, 132, 1)',
            pointHoverBackgroundColor: 'rgba(255, 99, 132, 1)',
            borderWidth: 2,
            data: [...zeroData],
            fill: false
          },
          {
            label: 'In Process',
            backgroundColor: 'transparent',
            borderColor: 'rgba(54, 162, 235, 1)',
            pointHoverBackgroundColor: 'rgba(54, 162, 235, 1)',
            borderWidth: 2,
            data: [...zeroData],
            fill: false
          }
        ]
      };

      this.currentRates = { publish: 0, deliver: 0, ack: 0 };
      this.currentGauges = { pending: 0, inProcess: 0 };
      this.metricsLoading = false;
    };

    if (!this.tenantCode) {
      buildEmptyMetrics();
      return;
    }

    this.tsdbMetricsService.getTSDBMetrics(this.tenantCode, queueCode, vnamespace, 5, startTime, endTime).subscribe({
      next: (result: any) => {
        if (!result || !result.datapoints || result.datapoints.length === 0) {
          buildEmptyMetrics();
          return;
        }

        const datapointMap = new Map<number, any>();
        result.datapoints.forEach((dp: any) => {
          datapointMap.set(dp.timestamp, dp);
        });

        const labels: string[] = [];
        const publishData: number[] = [];
        const deliveryData: number[] = [];
        const ackData: number[] = [];
        const pendingData: number[] = [];
        const inProcessData: number[] = [];

        const normalizedStartTime = Math.floor(startTime / 5) * 5;
        const normalizedEndTime = Math.floor(endTime / 5) * 5;

        let currentPending = 0;
        let currentInProcess = 0;

        for (let ts = normalizedStartTime; ts <= normalizedEndTime; ts += 5) {
          const date = new Date(ts * 1000);
          labels.push(`${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}:${date.getSeconds().toString().padStart(2, '0')}`);

          if (datapointMap.has(ts)) {
            const dp = datapointMap.get(ts);
            publishData.push(dp.published || 0);
            deliveryData.push(dp.delivered || 0);
            ackData.push(dp.acked || 0);

            currentPending = dp.pending !== undefined ? dp.pending : currentPending;
            currentInProcess = dp.inProcess !== undefined ? dp.inProcess : (dp.in_process !== undefined ? dp.in_process : currentInProcess);

            pendingData.push(currentPending);
            inProcessData.push(currentInProcess);
          } else {
            publishData.push(0);
            deliveryData.push(0);
            ackData.push(0);
            pendingData.push(currentPending);
            inProcessData.push(currentInProcess);
          }
        }

        this.metricsData = {
          labels: labels,
          datasets: [
            {
              label: 'Published',
              backgroundColor: 'rgba(51, 153, 255, 0.1)',
              borderColor: '#3399ff',
              pointHoverBackgroundColor: '#3399ff',
              borderWidth: 2,
              data: publishData,
              fill: true
            },
            {
              label: 'Delivered',
              backgroundColor: 'rgba(249, 177, 21, 0.1)',
              borderColor: '#f9b115',
              pointHoverBackgroundColor: '#f9b115',
              borderWidth: 2,
              data: deliveryData,
              fill: true
            },
            {
              label: 'Acked',
              backgroundColor: 'rgba(46, 184, 92, 0.1)',
              borderColor: '#2eb85c',
              pointHoverBackgroundColor: '#2eb85c',
              borderWidth: 2,
              data: ackData,
              fill: true
            }
          ]
        };

        this.gaugeMetricsData = {
          labels: labels,
          datasets: [
            {
              label: 'Pending',
              backgroundColor: 'transparent',
              borderColor: 'rgba(255, 99, 132, 1)',
              pointHoverBackgroundColor: 'rgba(255, 99, 132, 1)',
              borderWidth: 2,
              data: pendingData,
              fill: false
            },
            {
              label: 'In Process',
              backgroundColor: 'transparent',
              borderColor: 'rgba(54, 162, 235, 1)',
              pointHoverBackgroundColor: 'rgba(54, 162, 235, 1)',
              borderWidth: 2,
              data: inProcessData,
              fill: false
            }
          ]
        };

        const len = publishData.length;
        if (len > 0) {
          this.currentRates = {
            publish: publishData[len - 1] / 5,
            deliver: deliveryData[len - 1] / 5,
            ack: ackData[len - 1] / 5
          };
          this.currentGauges = {
            pending: pendingData[len - 1],
            inProcess: inProcessData[len - 1]
          };
        }
        this.metricsLoading = false;
      },
      error: (err) => {
        buildEmptyMetrics();
      }
    });
  }

  getExpirationInfo(queue: any): {
    show: boolean;
    isExpired: boolean;
    isNearExpiry: boolean;
    text: string;
    progressPercentage: number;
  } {
    if (!queue) {
      return { show: false, isExpired: false, isNearExpiry: false, text: '', progressPercentage: -1 };
    }

    const now = new Date();
    const expireAtRaw = queue.ExpireAt || queue.expireAt;

    if (expireAtRaw) {
      const expireAt = new Date(expireAtRaw);
      const isExpired = now > expireAt;

      if (isExpired) {
        return {
          show: true,
          isExpired: true,
          isNearExpiry: false,
          text: `Expired on ${expireAt.toLocaleString()}`,
          progressPercentage: 0
        };
      }

      const updatedAt = new Date(queue.UpdatedAt || queue.updatedAt || Date.now());
      const totalTime = expireAt.getTime() - updatedAt.getTime();
      const remainingTime = expireAt.getTime() - now.getTime();
      const progressPercentage = totalTime > 0 ? (remainingTime / totalTime) * 100 : 0;

      return {
        show: true,
        isExpired: false,
        isNearExpiry: progressPercentage <= 20,
        text: `Expires on ${expireAt.toLocaleString()}`,
        progressPercentage: Math.max(0, progressPercentage)
      };
    }

    const queueExpires = queue.QueueExpires || queue.queueExpires || 0;
    if (queueExpires > 0) {
      const updatedAt = new Date(queue.UpdatedAt || queue.updatedAt || Date.now());
      const expireAt = new Date(updatedAt.getTime() + (queueExpires * 1000));
      const isExpired = now > expireAt;

      if (isExpired) {
        return {
          show: true,
          isExpired: true,
          isNearExpiry: false,
          text: `Expired on ${expireAt.toLocaleString()}`,
          progressPercentage: 0
        };
      }

      const totalTime = queueExpires * 1000;
      const elapsedTime = now.getTime() - updatedAt.getTime();
      const remainingTime = totalTime - elapsedTime;
      const progressPercentage = totalTime > 0 ? (remainingTime / totalTime) * 100 : 0;

      return {
        show: true,
        isExpired: false,
        isNearExpiry: progressPercentage <= 20,
        text: `Expires on ${expireAt.toLocaleString()}`,
        progressPercentage: Math.max(0, progressPercentage)
      };
    }

    return { show: false, isExpired: false, isNearExpiry: false, text: '', progressPercentage: -1 };
  }

  getQueueTypeColor(type: string): string {
    const typeColors: { [key: string]: string } = {
      'standard': 'primary',
      'workflow_execution': 'primary',
      'workflow_activity': 'info'
    };
    return typeColors[type] || 'secondary';
  }

  getQueueStateColor(state: string): string {
    const stateColors: { [key: string]: string } = {
      'QueueActive': 'success',
      'active': 'success',
      'QueuePaused': 'warning',
      'paused': 'warning',
      'QueueDraining': 'info',
      'draining': 'info',
      'QueueStopped': 'danger',
      'stopped': 'danger'
    };
    return stateColors[state] || 'secondary';
  }

  getQueueStateLabel(state: string): string {
    const stateLabels: { [key: string]: string } = {
      'QueueActive': 'Active',
      'active': 'Active',
      'QueuePaused': 'Paused',
      'paused': 'Paused',
      'QueueDraining': 'Draining',
      'draining': 'Draining',
      'QueueStopped': 'Stopped',
      'stopped': 'Stopped'
    };
    return stateLabels[state] || state || 'Active';
  }

  getPriorityTypeColor(priorityType: string): string {
    const colors: { [key: string]: string } = {
      'normal': 'primary',
      'fair': 'success'
    };
    return colors[priorityType] || 'secondary';
  }

  getPriorityTypeLabel(priorityType: string): string {
    const labels: { [key: string]: string } = {
      'normal': 'Normal',
      'fair': 'Fair'
    };
    return labels[priorityType] || 'Unknown';
  }

  getCalculatedPriorityType(queue: any): string {
    if (!queue) return 'normal';
    const thresholds = queue.DesiredPriorityThresholds || queue.PriorityThresholds || queue.desiredPriorityThresholds || queue.priorityThresholds;
    if (thresholds && Object.keys(thresholds).length > 0) {
      const allValuesAreZero = Object.values(thresholds).every((value: any) => Number(value) === 0);
      return allValuesAreZero ? 'normal' : 'fair';
    }
    return 'normal';
  }

  getCalculatedMaxPriorityLevels(queue: any): number {
    if (!queue) return 1;
    const thresholds = queue.DesiredPriorityThresholds || queue.PriorityThresholds || queue.desiredPriorityThresholds || queue.priorityThresholds;
    if (thresholds) {
      if (typeof thresholds === 'object' && !Array.isArray(thresholds)) {
        return Object.keys(thresholds).length;
      } else if (Array.isArray(thresholds)) {
        return thresholds.length;
      }
    }
    return queue.MaxPriority || queue.maxPriority || 1;
  }

  getPriorityThresholdsArray(thresholds: any): number[] {
    if (!thresholds) return [];
    if (Array.isArray(thresholds)) return thresholds;
    if (typeof thresholds === 'object') {
      const result: number[] = [];
      Object.keys(thresholds).sort((a, b) => Number(a) - Number(b)).forEach(key => {
        result.push(thresholds[key]);
      });
      return result;
    }
    return [];
  }

  getHeadersArray(headers: { [key: string]: string }): { key: string, value: string }[] {
    if (!headers) return [];
    return Object.keys(headers).map(key => ({ key, value: headers[key] }));
  }
}

import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  CardModule,
  GridModule,
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { DashboardService } from './services/dashboard.service';
import { ChartjsModule } from '@coreui/angular-chartjs';
import { FormsModule } from '@angular/forms';
import { Subject, interval, Subscription } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { SpinnerComponent } from '@coreui/angular';

@Component({
  selector: 'app-dashboard',
  templateUrl: 'dashboard.component.html',
  styleUrls: ['dashboard.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    CardModule,
    GridModule,
    IconDirective,
    ChartjsModule,
    FormsModule,
    SpinnerComponent
  ]
})
export class DashboardComponent implements OnInit {
  dashboardSummary: any = null;
  dashboardLoading: boolean = false;
  dashboardError: boolean = false;

  metricsLoading: boolean = false;
  metricsData: any = {
    labels: [],
    datasets: []
  };
  currentRates: any = {
    publish: 0,
    deliver: 0,
    ack: 0
  };
  gaugeMetricsData: any = {
    labels: [],
    datasets: []
  };
  currentGauges: any = {
    pending: 0,
    inProcess: 0
  };
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
  selectedTimeRange: number = 600;
  private destroy$ = new Subject<void>();
  private refreshSubscription?: Subscription;

  constructor(private dashboardService: DashboardService) {}

  ngOnInit(): void {
    this.loadDashboardSummary();
    this.loadGlobalMetrics();

    // Auto-refresh every 5 seconds
    this.refreshSubscription = interval(5000)
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => {
        // Only refresh if we aren't currently loading to prevent overlapping requests
        if (!this.dashboardLoading) {
          this.loadDashboardSummary(true);
        }
        if (!this.metricsLoading) {
          this.loadGlobalMetrics(true);
        }
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
    if (this.refreshSubscription) {
      this.refreshSubscription.unsubscribe();
    }
  }

  loadDashboardSummary(isBackgroundRefresh: boolean = false): void {
    if (!isBackgroundRefresh) {
      this.dashboardLoading = true;
    }
    this.dashboardError = false;
    this.dashboardService.getDashboardSummary().subscribe({
      next: (response) => {
        this.dashboardSummary = response.result;
        this.dashboardLoading = false;
      },
      error: (error) => {
        console.error('Error loading dashboard summary:', error);
        if (!isBackgroundRefresh) {
          this.dashboardSummary = null;
        }
        this.dashboardLoading = false;
        this.dashboardError = true;
      }
    });
  }

  onTimeRangeChange(): void {
    this.loadGlobalMetrics();
  }

  loadGlobalMetrics(isBackgroundRefresh: boolean = false): void {
    if (!isBackgroundRefresh) {
      this.metricsLoading = true;
    }
    const endTime = Math.floor(Date.now() / 1000);
    const startTime = endTime - this.selectedTimeRange;
    
    this.dashboardService.getGlobalTSDBMetrics(5, startTime, endTime).subscribe({
      next: (response: any) => {
        this.metricsLoading = false;
        let result = response.result;
        if (!result || !result.datapoints) {
           result = { datapoints: [] }; // Handle as empty to pad with 0s
        }
        
        const datapointMap = new Map<number, any>();
        if (result.datapoints) {
          result.datapoints.forEach((dp: any) => {
            datapointMap.set(dp.timestamp, dp);
          });
        }

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
            currentInProcess = dp.inProcess !== undefined ? dp.inProcess : currentInProcess;
            
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
              backgroundColor: 'transparent',
              borderColor: 'rgba(50, 140, 255, 1)',
              pointBackgroundColor: 'rgba(50, 140, 255, 1)',
              data: publishData
            },
            {
              label: 'Delivered',
              backgroundColor: 'rgba(100, 255, 100, 0.1)',
              borderColor: 'rgba(100, 255, 100, 1)',
              pointBackgroundColor: 'rgba(100, 255, 100, 1)',
              data: deliveryData
            },
            {
              label: 'Acked',
              backgroundColor: 'transparent',
              borderColor: 'rgba(255, 200, 50, 1)',
              pointBackgroundColor: 'rgba(255, 200, 50, 1)',
              data: ackData
            }
          ]
        };

        this.gaugeMetricsData = {
          labels: labels,
          datasets: [
            {
              label: 'Pending (Wait in queue)',
              backgroundColor: 'transparent',
              borderColor: 'rgba(255, 99, 132, 1)',
              pointBackgroundColor: 'rgba(255, 99, 132, 1)',
              data: pendingData
            },
            {
              label: 'In Process (Claimed)',
              backgroundColor: 'transparent',
              borderColor: 'rgba(54, 162, 235, 1)',
              pointBackgroundColor: 'rgba(54, 162, 235, 1)',
              data: inProcessData
            }
          ]
        };

        let lastPublish = 0;
        let lastDeliver = 0;
        let lastAck = 0;
        
        if (result.datapoints && result.datapoints.length > 0) {
          const lastDp = result.datapoints[result.datapoints.length - 1];
          // Since each datapoint covers 5 seconds (the step is 5)
          lastPublish = (lastDp.published || 0) / 5;
          lastDeliver = (lastDp.delivered || 0) / 5;
          lastAck = (lastDp.acked || 0) / 5;
        }

        this.currentRates = {
          publish: lastPublish,
          deliver: lastDeliver,
          ack: lastAck
        };
        
        let lastPending = 0;
        let lastInProcess = 0;
        if (result.datapoints && result.datapoints.length > 0) {
          const lastDp = result.datapoints[result.datapoints.length - 1];
          lastPending = lastDp.pending || 0;
          lastInProcess = lastDp.inProcess || 0;
        }
        
        this.currentGauges = {
          pending: lastPending,
          inProcess: lastInProcess
        };
      },
      error: (error: any) => {
        console.error('Error loading global metrics:', error);
        this.metricsLoading = false;
        this.metricsData = { labels: [], datasets: [] };
        this.gaugeMetricsData = { labels: [], datasets: [] };
        this.currentRates = { publish: 0, deliver: 0, ack: 0 };
        this.currentGauges = { pending: 0, inProcess: 0 };
      }
    });
  }
}

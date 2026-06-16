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
import { Subject } from 'rxjs';
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

  constructor(private dashboardService: DashboardService) {}

  ngOnInit(): void {
    this.loadDashboardSummary();
    this.loadGlobalMetrics();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  loadDashboardSummary(): void {
    this.dashboardLoading = true;
    this.dashboardError = false;
    this.dashboardService.getDashboardSummary().subscribe({
      next: (response) => {
        this.dashboardSummary = response.result;
        this.dashboardLoading = false;
      },
      error: (error) => {
        console.error('Error loading dashboard summary:', error);
        this.dashboardSummary = null;
        this.dashboardLoading = false;
        this.dashboardError = true;
      }
    });
  }

  onTimeRangeChange(): void {
    this.loadGlobalMetrics();
  }

  loadGlobalMetrics(): void {
    this.metricsLoading = true;
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

        const normalizedStartTime = Math.floor(startTime / 5) * 5;
        const normalizedEndTime = Math.floor(endTime / 5) * 5;

        for (let ts = normalizedStartTime; ts <= normalizedEndTime; ts += 5) {
          const date = new Date(ts * 1000);
          labels.push(`${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}:${date.getSeconds().toString().padStart(2, '0')}`);
          
          if (datapointMap.has(ts)) {
            const dp = datapointMap.get(ts);
            publishData.push(dp.published || 0);
            deliveryData.push(dp.delivered || 0);
            ackData.push(dp.acked || 0);
          } else {
            publishData.push(0);
            deliveryData.push(0);
            ackData.push(0);
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
      },
      error: (error: any) => {
        console.error('Error loading global metrics:', error);
        this.metricsLoading = false;
        this.metricsData = { labels: [], datasets: [] };
      }
    });
  }
}

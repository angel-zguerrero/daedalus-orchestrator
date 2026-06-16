import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { Subject } from 'rxjs';
import { FormsModule } from '@angular/forms';
import { 
  CardModule, 
  GridModule, 
  ButtonModule,
  BadgeModule,
  TabDirective,
  TabPanelComponent,
  TabsComponent,
  TabsContentComponent,
  TabsListComponent
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { ExchangesComponent } from './exchanges/exchanges.component';
import { QueuesComponent } from './queues/queues.component';
import { BindingsComponent } from './bindings/bindings.component';
import { TenantsService } from '../services/tenants.service';
import { TSDBMetricsService } from '../services/tsdb-metrics.service';
import { ChartjsModule } from '@coreui/angular-chartjs';
import { SpinnerComponent } from '@coreui/angular';

@Component({
  selector: 'app-tenant-management',
  templateUrl: './tenant-management.component.html',
  styleUrls: ['./tenant-management.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    CardModule,
    GridModule,
    ButtonModule,
    BadgeModule,
    TabDirective,
    TabPanelComponent,
    FormsModule,
    TabsComponent,
    TabsContentComponent,
    TabsListComponent,
    IconDirective,
    ExchangesComponent,
    QueuesComponent,
    BindingsComponent,
    ChartjsModule,
    SpinnerComponent
  ]
})
export class TenantManagementComponent implements OnInit {
  tenantCode: string = '';
  tenantName: string = '';
  activeTab: string = 'summary';
  selectedTenant: any = null;
  tenantSummary: any = null;
  tenantSummaryLoading: boolean = false;
  tenantSummaryNotFound: boolean = false;

  // Metrics Data
  metricsLoading: boolean = false;
  metricsData: any = {
    datasets: [],
    labels: []
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
  private destroy$ = new Subject<void>();

  selectedTimeRange: number = 600; // Default to last 10 minutes

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private tenantsService: TenantsService,
    private tsdbMetricsService: TSDBMetricsService
  ) {}

  ngOnInit(): void {
    this.route.params.subscribe(params => {
      this.tenantCode = params['code'];
      // Load tenant details when tenant code is available
      if (this.tenantCode) {
        this.loadTenantDetails();
      }
    });

    this.route.queryParams.subscribe(queryParams => {
      if (queryParams['name']) {
        this.tenantName = queryParams['name'];
      }
      if (queryParams['tab']) {
        this.activeTab = queryParams['tab'];
      }
    });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  loadTenantSummary(): void {
    this.tenantSummaryLoading = true;
    this.tenantSummaryNotFound = false;
    this.tenantsService.getTenantSummary(this.tenantCode).subscribe({
      next: (response) => {
        this.tenantSummary = response.result;
        this.tenantSummaryLoading = false;
      },
      error: (error) => {
        console.error('Error loading tenant summary:', error);
        this.tenantSummary = null;
        this.tenantSummaryLoading = false;
        this.tenantSummaryNotFound = true;
      }
    });

    this.loadTenantMetrics();
  }

  onTimeRangeChange(): void {
    this.loadTenantMetrics();
  }

  loadTenantMetrics(): void {
    this.metricsLoading = true;
    const endTime = Math.floor(Date.now() / 1000);
    const startTime = endTime - this.selectedTimeRange;
    
    // Passing undefined for queueCode and vnamespace queries tenant-level scope
    this.tsdbMetricsService.getTSDBMetrics(this.tenantCode, undefined, undefined, 5, startTime, endTime).subscribe({
      next: (result: any) => {
        this.metricsLoading = false;
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
      error: (err: any) => {
        this.metricsLoading = false;
        this.metricsData = { labels: [], datasets: [] };
        this.gaugeMetricsData = { labels: [], datasets: [] };
        this.currentRates = { publish: 0, deliver: 0, ack: 0 };
        this.currentGauges = { pending: 0, inProcess: 0 };
        console.error('Failed to load global tenant metrics', err);
      }
    });
  }

  loadTenantDetails(): void {
    // Load tenant basic information
    this.tenantsService.getTenant(this.tenantCode).subscribe({
      next: (response) => {
        this.selectedTenant = response.result;
        if (!this.tenantName && response.result.Name) {
          this.tenantName = response.result.Name;
        }
      },
      error: (error) => {
        console.error('Error loading tenant details:', error);
      }
    });

    // Load tenant summary only if on summary tab
    if (this.activeTab === 'summary') {
      this.loadTenantSummary();
    }
  }

  navigateToTab(tabKey: string | number | undefined): void {
    if (tabKey && typeof tabKey === 'string') {
      this.activeTab = tabKey;
      this.router.navigate([], {
        relativeTo: this.route,
        queryParams: { tab: tabKey },
        queryParamsHandling: 'merge'
      });

      // Load tenant summary when Summary tab is selected
      if (tabKey === 'summary') {
        this.loadTenantSummary();
      }
    }
  }

  goBackToTenants(): void {
    this.router.navigate(['/tenants']);
  }
}

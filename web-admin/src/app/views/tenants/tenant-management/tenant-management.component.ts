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
      },
      error: (err: any) => {
        this.metricsLoading = false;
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

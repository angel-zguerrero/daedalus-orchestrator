import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { 
  CardModule, 
  GridModule, 
  ButtonModule,
  TabDirective,
  TabPanelComponent,
  TabsComponent,
  TabsContentComponent,
  TabsListComponent
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { ExchangesComponent } from './exchanges/exchanges.component';
import { QueuesComponent } from './queues/queues.component';
import { TenantsService } from '../services/tenants.service';

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
    TabDirective,
    TabPanelComponent,
    TabsComponent,
    TabsContentComponent,
    TabsListComponent,
    IconDirective,
    ExchangesComponent,
    QueuesComponent
  ]
})
export class TenantManagementComponent implements OnInit {
  tenantId: string = '';
  tenantName: string = '';
  activeTab: string = 'summary';
  selectedTenant: any = null;
  tenantSummary: any = null;
  tenantSummaryLoading: boolean = false;
  tenantSummaryNotFound: boolean = false;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private tenantsService: TenantsService
  ) {}

  ngOnInit(): void {
    this.route.params.subscribe(params => {
      this.tenantId = params['id'];
      // Load tenant details when tenant ID is available
      if (this.tenantId) {
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

  loadTenantSummary(): void {
    this.tenantSummaryLoading = true;
    this.tenantSummaryNotFound = false;
    this.tenantsService.getTenantSummary(this.tenantId).subscribe({
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
  }

  loadTenantDetails(): void {
    // Load tenant basic information
    this.tenantsService.getTenant(this.tenantId).subscribe({
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

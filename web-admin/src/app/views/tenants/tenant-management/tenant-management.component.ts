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
    ExchangesComponent
  ]
})
export class TenantManagementComponent implements OnInit {
  tenantId: string = '';
  tenantName: string = '';
  activeTab: string = 'summary';

  constructor(
    private route: ActivatedRoute,
    private router: Router
  ) {}

  ngOnInit(): void {
    this.route.params.subscribe(params => {
      this.tenantId = params['id'];
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

  navigateToTab(tabKey: string | number | undefined): void {
    if (tabKey && typeof tabKey === 'string') {
      this.activeTab = tabKey;
      this.router.navigate([], {
        relativeTo: this.route,
        queryParams: { tab: tabKey },
        queryParamsHandling: 'merge'
      });
    }
  }

  goBackToTenants(): void {
    this.router.navigate(['/tenants']);
  }
}

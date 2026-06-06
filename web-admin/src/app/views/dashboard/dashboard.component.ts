import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  CardModule,
  GridModule,
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { DashboardService } from './services/dashboard.service';

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
  ]
})
export class DashboardComponent implements OnInit {
  dashboardSummary: any = null;
  dashboardLoading: boolean = false;
  dashboardError: boolean = false;

  constructor(private dashboardService: DashboardService) {}

  ngOnInit(): void {
    this.loadDashboardSummary();
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
}

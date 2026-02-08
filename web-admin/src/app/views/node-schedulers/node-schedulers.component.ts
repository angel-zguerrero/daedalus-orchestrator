import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NodeSchedulersService } from './services/node-schedulers.service';
import { TableModule, UtilitiesModule, ButtonModule, ModalModule, CardModule, FormModule, GridModule, AlertComponent, SpinnerComponent, BadgeComponent } from '@coreui/angular';
import { ReactiveFormsModule, FormsModule } from '@angular/forms';
import { IconDirective } from '@coreui/icons-angular';
import { ErrorUtil } from '../../shared/utils/error.util';

@Component({
  selector: 'app-node-schedulers',
  templateUrl: './node-schedulers.component.html',
  styleUrls: ['./node-schedulers.component.scss'],
  standalone: true,
  imports: [
    AlertComponent,
    CommonModule,
    TableModule,
    UtilitiesModule,
    ButtonModule,
    ModalModule,
    CardModule,
    FormModule,
    GridModule,
    ReactiveFormsModule,
    FormsModule,
    SpinnerComponent,
    BadgeComponent,
    IconDirective
  ]
})
export class NodeSchedulersComponent implements OnInit {
  nodeSchedulers: any[] = [];
  cursor = '';
  cursors: string[] = [];
  pageSize = 20;
  searchQuery = '';

  public detailsModalVisible = false;
  public showAlert = false;
  public errorMessage = '';
  public loading = false;

  selectedNodeScheduler: any;

  constructor(
    private nodeSchedulersService: NodeSchedulersService
  ) { }

  ngOnInit(): void {
    this.cursors.push('')
    this.loadNodeSchedulers();
  }

  loadNodeSchedulers(cursor: string = '', isPrevious: boolean = false): void {
    if (!isPrevious && cursor) {
      this.cursors.push(cursor);
    }
    this.loading = true;
    this.nodeSchedulersService.getNodeSchedulers(cursor, this.pageSize, this.searchQuery).subscribe({
      next: (response) => {
        this.nodeSchedulers = response.result.Entities;
        this.cursor = response.result.Cursor;
        this.loading = false;
      },
      error: (error) => {
        this.showAlert = true;
        this.errorMessage = ErrorUtil.formatErrorMessage(error);
        this.loading = false;
      }
    });
  }

  searchNodeSchedulers(): void {
    this.cursors = [''];
    this.loadNodeSchedulers();
  }

  nextPage(): void {
    if (this.cursor) {
      this.loadNodeSchedulers(this.cursor);
    }
  }

  previousPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop()
      this.loadNodeSchedulers(this.cursors[this.cursors.length - 1], true);
    }
  }

  openDetailsModal(nodeScheduler: any): void {
    console.log('Selected node scheduler from table:', nodeScheduler);
    this.selectedNodeScheduler = nodeScheduler;
    this.detailsModalVisible = true;
    this.showAlert = false;
  }

  getRunningStatusColor(status: string): string {
    switch (status?.toLowerCase()) {
      case 'running':
        return 'success';
      case 'stopped':
        return 'warning';
      default:
        return 'secondary';
    }
  }

  getConnectionStatusColor(status: string): string {
    switch (status?.toLowerCase()) {
      case 'connected':
        return 'success';
      case 'disconnected':
        return 'danger';
      default:
        return 'secondary';
    }
  }

  formatBytes(bytes: number): string {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const remainingSeconds = seconds % 60;

    if (hours > 0) {
      return `${hours}h ${minutes}m ${remainingSeconds}s`;
    } else if (minutes > 0) {
      return `${minutes}m ${remainingSeconds}s`;
    } else {
      return `${remainingSeconds}s`;
    }
  }

  getAdditionalInfo(): { key: string, value: string }[] {
    if (!this.selectedNodeScheduler?.Information) {
      return [];
    }

    const standardKeys = ['CPU', 'Memory', 'Disk', 'OS'];
    const additionalInfo: { key: string, value: string }[] = [];

    Object.keys(this.selectedNodeScheduler.Information).forEach(key => {
      if (!standardKeys.includes(key)) {
        additionalInfo.push({
          key: key,
          value: this.selectedNodeScheduler.Information[key]
        });
      }
    });

    return additionalInfo;
  }
}

import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { JobWorkersService } from './services/job-workers.service';
import { TableModule, UtilitiesModule, ButtonModule, ModalModule, CardModule, FormModule, GridModule, AlertComponent, SpinnerComponent, BadgeComponent } from '@coreui/angular';
import { ReactiveFormsModule, FormsModule } from '@angular/forms';
import { IconDirective } from '@coreui/icons-angular';
import { ErrorUtil } from '../../shared/utils/error.util';

@Component({
    selector: 'app-job-workers',
    templateUrl: './job-workers.component.html',
    styleUrls: ['./job-workers.component.scss'],
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
export class JobWorkersComponent implements OnInit {
    jobWorkers: any[] = [];
    cursor = '';
    cursors: string[] = [];
    pageSize = 20;
    searchQuery = '';

    public detailsModalVisible = false;
    public showAlert = false;
    public errorMessage = '';
    public loading = false;

    selectedJobWorker: any;

    constructor(
        private jobWorkersService: JobWorkersService
    ) { }

    ngOnInit(): void {
        this.cursors.push('')
        this.loadJobWorkers();
    }

    loadJobWorkers(cursor: string = '', isPrevious: boolean = false): void {
        if (!isPrevious && cursor) {
            this.cursors.push(cursor);
        }
        this.loading = true;
        this.jobWorkersService.getJobWorkers(cursor, this.pageSize, this.searchQuery).subscribe({
            next: (response) => {
                this.jobWorkers = response.result.Entities;
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

    searchJobWorkers(): void {
        this.cursors = [''];
        this.loadJobWorkers();
    }

    nextPage(): void {
        if (this.cursor) {
            this.loadJobWorkers(this.cursor);
        }
    }

    previousPage(): void {
        if (this.cursors.length > 1) {
            this.cursors.pop()
            this.loadJobWorkers(this.cursors[this.cursors.length - 1], true);
        }
    }

    openDetailsModal(jobWorker: any): void {
        this.selectedJobWorker = jobWorker;
        this.detailsModalVisible = true;
        this.showAlert = false;
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

    getAdditionalInfo(): { key: string, value: string }[] {
        if (!this.selectedJobWorker?.Information) {
            return [];
        }

        const standardKeys = ['CPU', 'Memory', 'Disk', 'OS'];
        const additionalInfo: { key: string, value: string }[] = [];

        Object.keys(this.selectedJobWorker.Information).forEach(key => {
            if (!standardKeys.includes(key)) {
                additionalInfo.push({
                    key: key,
                    value: this.selectedJobWorker.Information[key]
                });
            }
        });

        return additionalInfo;
    }

    getCapacityPolicies(): { key: string, value: any }[] {
        if (!this.selectedJobWorker?.ClaimWorkCapacityPolicies) {
            return [];
        }
        return Object.keys(this.selectedJobWorker.ClaimWorkCapacityPolicies).map(key => ({
            key: key,
            value: this.selectedJobWorker.ClaimWorkCapacityPolicies[key]
        }));
    }
}

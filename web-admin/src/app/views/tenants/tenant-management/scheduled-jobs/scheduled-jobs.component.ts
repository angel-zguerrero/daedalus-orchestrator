import { Component, OnInit, Input } from '@angular/core';
import { CommonModule, AsyncPipe, DatePipe } from '@angular/common';
import { ScheduledJobsService } from '../services/scheduled-jobs.service';
import { VNamespacesService } from '../services/vnamespaces.service';
import {
  TableModule,
  UtilitiesModule,
  ButtonModule,
  ModalModule,
  CardModule,
  FormModule,
  GridModule,
  AlertComponent,
  SpinnerComponent,
  BadgeComponent,
  NavModule,
  TabsModule
} from '@coreui/angular';
import { ReactiveFormsModule, FormsModule, FormBuilder, FormGroup, Validators, FormControl } from '@angular/forms';
import { IconDirective } from '@coreui/icons-angular';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { Observable } from 'rxjs';
import { startWith, debounceTime, switchMap, map } from 'rxjs/operators';
import { ErrorUtil } from '../../../../shared/utils/error.util';

export interface ScheduledJob {
  ID: string;
  Code?: string;
  TenantID: string;
  TargetType: string;
  TargetID: string;
  TargetCode: string;
  RoutingKeyOrPatternOrQueueCode: string;
  VNamespace: string;
  Content: string;
  ContentType: string;
  Headers?: { [key: string]: string };
  Handler?: string;
  Parameters?: { [key: string]: string };
  Priority: number;
  State: string;
  Type: string;
  Every?: string;
  CronExpression?: string;
  RunAt?: string;
  RunAfter?: string;
  NextRunAt: string;
  CreatedAt: string;
  UpdatedAt: string;
}

@Component({
  selector: 'app-scheduled-jobs',
  templateUrl: './scheduled-jobs.component.html',
  styleUrls: ['./scheduled-jobs.component.scss'],
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
    IconDirective,
    MatFormFieldModule,
    MatInputModule,
    MatAutocompleteModule,
    AsyncPipe,
    DatePipe,
    NavModule,
    TabsModule
  ]
})
export class ScheduledJobsComponent implements OnInit {
  @Input() tenantCode: string = '';

  scheduledJobs: ScheduledJob[] = [];
  cursor = '';
  cursors: string[] = [];
  pageSize = 20;

  public createModalVisible = false;
  public deleteModalVisible = false;
  public detailsModalVisible = false;

  public showAlert = false;
  public errorMessage = '';
  public successMessage = '';
  public loading = false;

  selectedJob: ScheduledJob | null = null;
  activeCreateTab: 'one-off' | 'recurring' = 'one-off';

  oneOffForm: FormGroup;
  recurringForm: FormGroup;

  vnamespaceCtrl = new FormControl('');
  filteredVNamespaces: Observable<any[]>;

  vnamespaceFilterCtrl = new FormControl('');
  filteredFilterVNamespaces: Observable<any[]>;
  selectedVNamespaceFilter = '';

  constructor(
    private scheduledJobsService: ScheduledJobsService,
    private vNamespacesService: VNamespacesService,
    private fb: FormBuilder
  ) {
    this.oneOffForm = this.fb.group({
      targetType: ['queue', Validators.required],
      targetCode: ['', Validators.required],
      vnamespace: this.vnamespaceCtrl,
      content: [''],
      contentType: ['text/plain'],
      handler: [''],
      priority: [0],
      runAt: [''],
      runAfter: ['5m']
    });

    this.recurringForm = this.fb.group({
      targetType: ['queue', Validators.required],
      targetCode: ['', Validators.required],
      vnamespace: this.vnamespaceCtrl,
      content: [''],
      contentType: ['text/plain'],
      handler: [''],
      priority: [0],
      every: ['5m'],
      cronExpression: ['']
    });

    this.filteredVNamespaces = this.vnamespaceCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(300),
      switchMap(value => this._filterVNamespaces(value || ''))
    );

    this.filteredFilterVNamespaces = this.vnamespaceFilterCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(300),
      switchMap(value => this._filterVNamespaces(value || ''))
    );
  }

  ngOnInit(): void {
    if (this.tenantCode) {
      this.cursors.push('');
      this.loadScheduledJobs();
    }
  }

  private _filterVNamespaces(value: string): Observable<any[]> {
    return this.vNamespacesService.getVNamespaces(this.tenantCode, '', 20, value).pipe(
      map(response => response.data || [])
    );
  }

  loadScheduledJobs(cursor: string = '', isPrevious: boolean = false): void {
    if (!isPrevious && cursor) {
      this.cursors.push(cursor);
    }
    this.loading = true;

    this.scheduledJobsService.getScheduledJobs(
      this.tenantCode,
      cursor,
      this.pageSize,
      this.selectedVNamespaceFilter
    ).subscribe({
      next: (response) => {
        this.scheduledJobs = response.result.Entities || [];
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

  onVNamespaceFilterChange(value: string): void {
    this.selectedVNamespaceFilter = value;
    this.cursors = [''];
    this.loadScheduledJobs();
  }

  nextPage(): void {
    if (this.cursor) {
      this.loadScheduledJobs(this.cursor);
    }
  }

  previousPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop();
      this.loadScheduledJobs(this.cursors[this.cursors.length - 1], true);
    }
  }

  openCreateModal(): void {
    this.createModalVisible = true;
    this.activeCreateTab = 'one-off';
    this.oneOffForm.reset({
      targetType: 'queue',
      targetCode: '',
      content: '',
      contentType: 'text/plain',
      handler: '',
      priority: 0,
      runAt: '',
      runAfter: '5m'
    });
    this.recurringForm.reset({
      targetType: 'queue',
      targetCode: '',
      content: '',
      contentType: 'text/plain',
      handler: '',
      priority: 0,
      every: '5m',
      cronExpression: ''
    });
    this.showAlert = false;
  }

  openDeleteModal(job: ScheduledJob): void {
    this.selectedJob = job;
    this.deleteModalVisible = true;
  }

  openDetailsModal(job: ScheduledJob): void {
    this.selectedJob = job;
    this.detailsModalVisible = true;
  }

  createOneOffJob(): void {
    if (this.oneOffForm.invalid) {
      this.oneOffForm.markAllAsTouched();
      return;
    }

    const formValue = this.oneOffForm.value;
    const payload = {
      ...formValue,
      vnamespace: this.vnamespaceCtrl.value || 'default'
    };

    this.loading = true;
    this.scheduledJobsService.createOneOffScheduledJob(this.tenantCode, payload).subscribe({
      next: () => {
        this.loading = false;
        this.createModalVisible = false;
        this.loadScheduledJobs();
        this.showAlert = false;
      },
      error: (error) => {
        this.loading = false;
        this.showAlert = true;
        this.errorMessage = ErrorUtil.formatErrorMessage(error);
      }
    });
  }

  createRecurringJob(): void {
    if (this.recurringForm.invalid) {
      this.recurringForm.markAllAsTouched();
      return;
    }

    const formValue = this.recurringForm.value;
    const payload = {
      ...formValue,
      vnamespace: this.vnamespaceCtrl.value || 'default'
    };

    this.loading = true;
    this.scheduledJobsService.createRecurringScheduledJob(this.tenantCode, payload).subscribe({
      next: () => {
        this.loading = false;
        this.createModalVisible = false;
        this.loadScheduledJobs();
        this.showAlert = false;
      },
      error: (error) => {
        this.loading = false;
        this.showAlert = true;
        this.errorMessage = ErrorUtil.formatErrorMessage(error);
      }
    });
  }

  deleteJob(): void {
    if (!this.selectedJob) return;

    this.loading = true;
    this.scheduledJobsService.deleteScheduledJob(this.tenantCode, this.selectedJob.ID).subscribe({
      next: () => {
        this.loading = false;
        this.deleteModalVisible = false;
        this.selectedJob = null;
        this.loadScheduledJobs();
      },
      error: (error) => {
        this.loading = false;
        this.showAlert = true;
        this.errorMessage = ErrorUtil.formatErrorMessage(error);
      }
    });
  }

  getStateBadgeColor(state: string): string {
    switch (state) {
      case 'idle': return 'info';
      case 'delivered': return 'success';
      default: return 'secondary';
    }
  }

  getTypeBadgeColor(type: string): string {
    switch (type) {
      case 'OneOff': return 'warning';
      case 'Recurring': return 'primary';
      default: return 'secondary';
    }
  }

  getScheduleDisplay(job: ScheduledJob): string {
    if (job.Type === 'OneOff') {
      if (job.RunAfter) return `After ${job.RunAfter}`;
      if (job.RunAt) return `At ${job.RunAt}`;
      return 'Immediate';
    }
    if (job.Type === 'Recurring') {
      if (job.Every) return `Every ${job.Every}`;
      if (job.CronExpression) return `Cron: ${job.CronExpression}`;
    }
    return '-';
  }
}

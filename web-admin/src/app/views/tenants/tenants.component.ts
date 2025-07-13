import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TenantsService } from './services/tenants.service';
import { TableModule, UtilitiesModule, ButtonModule, ModalModule, CardModule, FormModule, GridModule, AlertComponent, SpinnerComponent } from '@coreui/angular';
import { ReactiveFormsModule, FormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import * as XLSX from 'xlsx';

@Component({
  selector: 'app-tenants',
  templateUrl: './tenants.component.html',
  styleUrls: ['./tenants.component.scss'],
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
    SpinnerComponent
  ]
})
export class TenantsComponent implements OnInit {
  tenants: any[] = [];
  cursor = '';
  cursors: string[] = [];
  pageSize = 20;

  public createModalVisible = false;
  public editModalVisible = false;
  public deleteModalVisible = false;
  public detailsModalVisible = false;
  public bulkUploadModalVisible = false;

  public showAlert = false;
  public errorMessage = '';
  public loading = false;

  tenantForm: FormGroup;
  tenantFormUpdate: FormGroup;
  selectedTenant: any;

  constructor(
    private tenantsService: TenantsService,
    private fb: FormBuilder
  ) {
    this.tenantForm = this.fb.group({
      name: ['', Validators.required],
      code: ['', Validators.required]
    });
    this.tenantFormUpdate = this.fb.group({
      name: ['', Validators.required],
      code: [{ value: '', disabled: true }]
    });
  }

  ngOnInit(): void {
    this.cursors.push('')
    this.loadTenants();
  }

  loadTenants(cursor: string = '', isPrevious: boolean = false): void {
    if (!isPrevious && cursor) {
      this.cursors.push(cursor);
    }
    this.tenantsService.getTenants(cursor, this.pageSize).subscribe({
      next: (response) => {
        this.tenants = response.result.Entities;
        this.cursor = response.result.Cursor;
      },
      error: (error) => {
        this.showAlert = true;
        this.errorMessage = error.error;
      }
    });
  }

  nextPage(): void {
    if (this.cursor) {
      this.loadTenants(this.cursor);
    }
  }

  previousPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop()
      this.loadTenants(this.cursors[this.cursors.length - 1], true);
    }
  }

  openCreateModal(): void {
    this.createModalVisible = true;
    this.tenantForm.reset();
  }

  openEditModal(tenant: any): void {
    this.selectedTenant = tenant;
    this.tenantFormUpdate.reset();
    this.tenantFormUpdate.patchValue({
      name: tenant.Name,
      code: tenant.Code
    });
    this.editModalVisible = true;
  }

  openDeleteModal(tenant: any): void {
    this.selectedTenant = tenant;
    this.deleteModalVisible = true;
  }

  openDetailsModal(tenant: any): void {
    this.tenantsService.getTenant(tenant.ID).subscribe({
      next: (response) => {
        this.selectedTenant = response.result;
        this.detailsModalVisible = true;
      },
      error: (error) => {
        this.showAlert = true;
        this.errorMessage = error.error;
      }
    });
  }

  createTenant(): void {
    if (this.tenantForm.valid) {
      this.tenantsService.assertTenant(this.tenantForm.value).subscribe({
        next: () => {
          this.createModalVisible = false;
          this.loadTenants();
          this.showAlert = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = error.error;
        }
      });
    }
  }
  updateTenant(): void {
    if (this.tenantFormUpdate.valid) {
      const tenantData = this.tenantFormUpdate.getRawValue();
      this.tenantsService.assertTenant(tenantData).subscribe({
        next: () => {
          this.editModalVisible = false;
          this.loadTenants();
          this.showAlert = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = error.error;
        }
      });
    }
  }

  deleteTenant(): void {
    this.tenantsService.deleteTenant(this.selectedTenant.ID).subscribe(() => {
      this.deleteModalVisible = false;
      this.loadTenants();
    });
  }

  openBulkUploadModal(): void {
    this.bulkUploadModalVisible = true;
  }

  private file: File | null = null;

  onFileChange(event: any): void {
    this.file = event.target.files[0];
  }

  uploadTenants(): void {
    if (!this.file) {
      this.showAlert = true;
      this.errorMessage = 'Please select a file to upload.';
      return;
    }

    this.loading = true;
    const fileReader = new FileReader();
    fileReader.onload = (e: any) => {
      const data = new Uint8Array(e.target.result);
      const workbook = XLSX.read(data, { type: 'array' });
      const worksheet = workbook.Sheets[workbook.SheetNames[0]];
      const tenants = XLSX.utils.sheet_to_json(worksheet, { header: ['Name', 'Code'] });

      // Remove header row
      tenants.shift();

      if (tenants.length === 0) {
        this.showAlert = true;
        this.errorMessage = 'The uploaded file is empty.';
        return;
      }

      this.tenantsService.bulkAssertTenants({ tenants }).subscribe({
        next: () => {
          this.bulkUploadModalVisible = false;
          this.loadTenants();
          this.showAlert = false;
          this.loading = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = error.error;
          this.loading = false;
        }
      });
    };
    fileReader.readAsArrayBuffer(this.file);
  }
}

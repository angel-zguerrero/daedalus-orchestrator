import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TenantsService } from './services/tenants.service';
import { TableModule, UtilitiesModule, ButtonModule, ModalModule, CardModule, FormModule, GridModule } from '@coreui/angular';
import { ReactiveFormsModule, FormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';

@Component({
  selector: 'app-tenants',
  templateUrl: './tenants.component.html',
  styleUrls: ['./tenants.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    TableModule,
    UtilitiesModule,
    ButtonModule,
    ModalModule,
    CardModule,
    FormModule,
    GridModule,
    ReactiveFormsModule,
    FormsModule
  ]
})
export class TenantsComponent implements OnInit {
  tenants: any[] = [];
  cursor = '';
  cursors: string[] = [];
  pageSize = 10;

  public createModalVisible = false;
  public editModalVisible = false;
  public deleteModalVisible = false;
  public detailsModalVisible = false;

  tenantForm: FormGroup;
  selectedTenant: any;

  constructor(
    private tenantsService: TenantsService,
    private fb: FormBuilder
  ) {
    this.tenantForm = this.fb.group({
      name: ['', Validators.required],
      code: ['', Validators.required]
    });
  }

  ngOnInit(): void {
    this.loadTenants();
  }

  loadTenants(cursor: string = '', isPrevious: boolean = false): void {
    if (!isPrevious) {
      this.cursors.push(this.cursor);
    }

    this.tenantsService.getTenants(cursor, this.pageSize).subscribe(response => {
      this.tenants = response.result.Entities;
      this.cursor = response.result.Cursor;
    });
  }

  nextPage(): void {
    if (this.cursor) {
      this.loadTenants(this.cursor);
    }
  }

  previousPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop(); // Remove current cursor
      const previousCursor = this.cursors.pop() || ''; // Get previous cursor
      this.loadTenants(previousCursor, true);
    }
  }

  openCreateModal(): void {
    this.createModalVisible = true;
    this.tenantForm.reset();
  }

  openEditModal(tenant: any): void {
    this.selectedTenant = tenant;
    this.tenantForm.patchValue({
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
    this.tenantsService.getTenant(tenant.ID).subscribe(response => {
      this.selectedTenant = response.result;
      this.detailsModalVisible = true;
    });
  }

  createTenant(): void {
    if (this.tenantForm.valid) {
      this.tenantsService.createTenant(this.tenantForm.value).subscribe(() => {
        this.createModalVisible = false;
        this.loadTenants();
      });
    }
  }

  updateTenant(): void {
    if (this.tenantForm.valid) {
      this.tenantsService.updateTenant(this.selectedTenant.ID, this.tenantForm.value).subscribe(() => {
        this.editModalVisible = false;
        this.loadTenants();
      });
    }
  }

  deleteTenant(): void {
    this.tenantsService.deleteTenant(this.selectedTenant.ID).subscribe(() => {
      this.deleteModalVisible = false;
      this.loadTenants();
    });
  }
}

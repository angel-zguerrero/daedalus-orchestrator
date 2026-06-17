import { NgTemplateOutlet, NgIf } from '@angular/common';
import { Component, inject, input } from '@angular/core';
import { AuthService } from '../../../auth/auth.service';

import {
  AvatarComponent,
  BreadcrumbRouterComponent,
  ContainerComponent,
  DropdownComponent,
  DropdownItemDirective,
  DropdownMenuDirective,
  DropdownToggleDirective,
  HeaderComponent,
  HeaderNavComponent,
  HeaderTogglerDirective,
  SidebarToggleDirective
} from '@coreui/angular';

import { IconDirective } from '@coreui/icons-angular';

@Component({
  selector: 'app-default-header',
  templateUrl: './default-header.component.html',
  styleUrl: './default-header.component.scss',
  imports: [
    ContainerComponent,
    HeaderTogglerDirective,
    SidebarToggleDirective,
    IconDirective,
    HeaderNavComponent,
    NgTemplateOutlet,
    NgIf,
    BreadcrumbRouterComponent,
    DropdownComponent,
    DropdownToggleDirective,
    AvatarComponent,
    DropdownMenuDirective,
    DropdownItemDirective
  ]
})
export class DefaultHeaderComponent extends HeaderComponent {
  readonly authService = inject(AuthService); // Changed to public for HTML access
  username: string | null = null;

  constructor() {
    super();
  }

  ngOnInit() {
    this.username = this.authService.getLoggedInUsername();
  }

  sidebarId = input('sidebar1');

  logout() {
    this.authService.logout();
  }
}

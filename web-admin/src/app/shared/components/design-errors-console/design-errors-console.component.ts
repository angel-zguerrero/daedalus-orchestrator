import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-design-errors-console',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './design-errors-console.component.html',
  styleUrls: ['./design-errors-console.component.scss']
})
export class DesignErrorsConsoleComponent {
  @Input() errors: string[] = [];
  @Input() mode: 'bottom-bar' | 'table-row' = 'bottom-bar';
  @Input() title: string = 'Workflow Design Errors';

  public isCollapsed: boolean = false;

  public toggleCollapse(): void {
    this.isCollapsed = !this.isCollapsed;
  }
}

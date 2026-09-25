import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    data: {
      title: 'Activity Templates'
    },
    children: [
      {
        path: '',
        loadComponent: () =>
          import('./activity-templates.component').then((m) => m.ActivityTemplatesComponent),
        data: {
          title: ''
        }
      },
      {
        path: 'new',
        loadComponent: () =>
          import('./activity-template-editor/activity-template-editor.component').then(
            (m) => m.ActivityTemplateEditorComponent
          ),
        data: {
          title: 'Create Activity Template'
        }
      },
      {
        path: ':id/edit',
        loadComponent: () =>
          import('./activity-template-editor/activity-template-editor.component').then(
            (m) => m.ActivityTemplateEditorComponent
          ),
        data: {
          title: 'Edit Activity Template'
        }
      }
    ]
  }
];

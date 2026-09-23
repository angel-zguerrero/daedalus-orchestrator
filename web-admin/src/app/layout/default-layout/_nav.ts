import { INavData } from '@coreui/angular';

export const navItems: INavData[] = [
  {
    name: 'Dashboard',
    url: '/dashboard',
    iconComponent: { name: 'cil-speedometer' }
  },
  {
    name: 'Workflows',
    url: '/workflows',
    iconComponent: { name: 'cil-transfer' }
  },
  {
    name: 'Activity Templates',
    url: '/activity-templates',
    iconComponent: { name: 'cil-puzzle' }
  },
  {
    name: 'Tenants',
    url: '/tenants',
    iconComponent: { name: 'cil-people' }
  },
  {
    name: 'Configs & Secrets',
    url: '/configs-secrets',
    iconComponent: { name: 'cil-lock-locked' }
  },
  {
    name: 'Service Accounts',
    url: '/service-accounts',
    iconComponent: { name: 'cil-shield-alt' }
  },
  {
    name: 'Cluster Management',
    url: '/cluster',
    iconComponent: { name: 'cil-layers' }
  },
  {
    name: 'Job Workers',
    url: '/job-workers',
    iconComponent: { name: 'cil-list' }
  },
  {
    name: 'Users',
    url: '/users',
    iconComponent: { name: 'cil-user' }
  }
];

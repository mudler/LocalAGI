import { createBrowserRouter } from 'react-router-dom';
import App from './App';
import Home from './pages/Home';
import AgentSettings from './pages/AgentSettings';
import AgentsList from './pages/AgentsList';
import CreateAgent from './pages/CreateAgent';
import Chat from './pages/Chat';
import ActionsPlayground from './pages/ActionsPlayground';
import GroupCreate from './pages/GroupCreate';

// Get the base URL from Vite's environment variables or default to '/app/'
const BASE_URL = import.meta.env.BASE_URL || '/app';

// Create a router with the base URL
export const router = createBrowserRouter([
  {
    path: '/',
    element: <App />,
    children: [
      {
        index: true,
        element: <Home />
      },
      {
        path: 'agents',
        element: <AgentsList />
      },
      {
        path: 'create',
        element: <CreateAgent />
      },
      {
        path: 'settings/:name',
        element: <AgentSettings />
      },
      {
        path: 'talk/:name',
        element: <Chat />
      },
      {
        path: 'actions-playground',
        element: <ActionsPlayground />
      },
      {
        path: 'group-create',
        element: <GroupCreate />
      }
    ]
  }
], {
  basename: BASE_URL // Set the base URL for all routes
});

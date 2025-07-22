import { createBrowserRouter } from "react-router-dom";
import App from "./App";
import Home from "./pages/Home";
import AgentSettings from "./pages/AgentSettings";
import AgentsList from "./pages/AgentsList";
import CreateAgent from "./pages/CreateAgent";
import Chat from "./pages/Chat";
import ActionsPlayground from "./pages/ActionsPlayground";
import GroupCreate from "./pages/GroupCreate";
import AgentStatus from "./pages/AgentStatus";
import ImportAgent from "./pages/ImportAgent";
import Usage from "./pages/Usage";
import ProtectedRoute from "./components/ProtectedRoute";

const BASE_URL = import.meta.env.BASE_URL || "/app";

export const router = createBrowserRouter(
  [
    {
      path: "/",
      element: <App />,
      children: [
        {
          index: true,
          element: <Home />,
        },
        {
          path: "agents",
          element: (
            <ProtectedRoute>
              <AgentsList />
            </ProtectedRoute>
          ),
        },
        {
          path: "create",
          element: (
            <ProtectedRoute>
              <CreateAgent />
            </ProtectedRoute>
          ),
        },
        {
          path: "settings/:id",
          element: (
            <ProtectedRoute>
              <AgentSettings />
            </ProtectedRoute>
          ),
        },
        {
          path: "talk/:id",
          element: (
            <ProtectedRoute>
              <Chat />
            </ProtectedRoute>
          ),
        },
        {
          path: "actions-playground",
          element: (
            <ProtectedRoute>
              <ActionsPlayground />
            </ProtectedRoute>
          ),
        },
        {
          path: "group-create",
          element: (
            <ProtectedRoute>
              <GroupCreate />
            </ProtectedRoute>
          ),
        },
        {
          path: "import",
          element: (
            <ProtectedRoute>
              <ImportAgent />
            </ProtectedRoute>
          ),
        },
        {
          path: "status/:id",
          element: (
            <ProtectedRoute>
              <AgentStatus />
            </ProtectedRoute>
          ),
        },
        {
          path: "usage",
          element: (
            <ProtectedRoute>
              <Usage />
            </ProtectedRoute>
          ),
        },
      ],
    },
  ],
  {
    basename: BASE_URL,
  }
);

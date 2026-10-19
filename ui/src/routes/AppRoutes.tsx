import { type PropsWithChildren } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { LoginPage } from "../pages/auth/LoginPage";
import { RegisterPage } from "../pages/auth/RegisterPage";
import { useSession } from "../hooks/useSession";
import { useQuery } from "@tanstack/react-query";
import { authService } from "../services/auth";

import { QueryErrorView } from "../components/QueryErrorView";
import { DashboardLayout } from "../layouts/DashboardLayout";
import { ServicesPage } from "../pages/internal/ServicesPage";
import { ServiceDetailPage } from "../pages/internal/ServiceDetailPage";
import { ProxyPage } from "../pages/internal/ProxyPage";
import { ProxyDetailsPage } from "../pages/internal/ProxyDetailsPage";
import { SettingsPage } from "../pages/internal/SettingsPage";
import { OperationsPage } from "../pages/internal/OperationsPage";
import { StatusPage } from "../pages/internal/StatusPage";

const PrivateRoute = ({
  children,
  redirectTo,
}: PropsWithChildren<{ redirectTo: string }>) => {
  const { user } = useSession();
  return user ? children : <Navigate to={redirectTo} replace />;
};

const PublicRoute = ({ children }: PropsWithChildren) => {
  const { user } = useSession();
  return !user ? children : <Navigate to="/" replace />;
};

export const AppRoutes = () => {
  const adminExistsQuery = useQuery<boolean>({
    queryKey: ["admin-created"],
    queryFn: async ({ signal }) => {
      const setupDone = await authService.isInitialSetupDone(signal);
      if (!setupDone) await authService.logout();
      return setupDone;
    },
  });

  if (adminExistsQuery.isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <span className="text-gray-500 text-lg animate-pulse">Loading...</span>
      </div>
    );
  }

  if (adminExistsQuery.isError)
    return (
      <QueryErrorView
        error={adminExistsQuery.error}
        onRetry={adminExistsQuery.refetch}
      />
    );

  if (adminExistsQuery.data == null) {
    return (
      <QueryErrorView
        error={new Error("Unexpected empty response from server.")}
        onRetry={adminExistsQuery.refetch}
      />
    );
  }
  const isSetupDone = adminExistsQuery.data;
  return (
    <BrowserRouter>
      <Routes>
        {isSetupDone ? (
          <Route
            path="/login"
            element={
              <PublicRoute>
                <LoginPage />
              </PublicRoute>
            }
          />
        ) : (
          <Route
            path="/register"
            element={
              <PublicRoute>
                <RegisterPage refresh={adminExistsQuery.refetch} />
              </PublicRoute>
            }
          />
        )}
        <Route
          path="/"
          element={
            <PrivateRoute redirectTo={isSetupDone ? "/login" : "/register"}>
              <DashboardLayout />
            </PrivateRoute>
          }
        >
          <Route index element={<ServicesPage />} />
          <Route path="/proxy" element={<ProxyPage />} />
          <Route path="/operations" element={<OperationsPage />} />
          <Route path="/status" element={<StatusPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="/services/:service_id" element={<ServiceDetailPage />} />
          <Route path="/proxy/:proxy_id" element={<ProxyDetailsPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
};

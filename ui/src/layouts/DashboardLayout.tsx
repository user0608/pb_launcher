import { useMemo, type PropsWithChildren } from "react";
import { NavLink, Outlet, useLocation } from "react-router-dom";
import {
  Activity,
  History,
  Server,
  Waypoints,
  User,
  LogOut,
  Settings,
} from "lucide-react";
import { useConfirmModal } from "../hooks/useConfirmModal";
import { authService } from "../services/auth";
import { useViewportHeight } from "../hooks/useViewportHeight";
import classNames from "classnames";

export const DASHBOARD_LAYOUT_APP_BAR_HEIGHT = 56;

export const DashboardLayout = ({ children }: PropsWithChildren) => {
  const { pathname } = useLocation();
  const height = useViewportHeight();

  const confirm = useConfirmModal();
  const selected = useMemo(() => {
    if (pathname === "/" || pathname.startsWith("/services/")) return "service";
    if (pathname === "/proxy" || pathname.startsWith("/proxy/")) return "proxy";
    if (pathname === "/operations") return "operations";
    if (pathname === "/status") return "status";
    if (pathname === "/settings") return "settings";
    return "<none>";
  }, [pathname]);

  const logout = async () => {
    const confirmed = await confirm(
      "Sign out",
      "Are you sure you want to sign out?",
    );
    if (!confirmed) return;
    await authService.logout();
  };

  const closeDropdown = () => {
    (document.activeElement as HTMLElement)?.blur?.();
  };

  return (
    <div style={{ height }} className="bg-base-200 flex flex-col items-center">
      <header
        style={{ height: DASHBOARD_LAYOUT_APP_BAR_HEIGHT }}
        className="w-full bg-base-100 shadow-sm"
      >
        <div className="mx-auto flex w-full items-center justify-between gap-2 px-2 py-3 sm:px-4">
          <nav className="flex min-w-0 flex-1 overflow-x-auto">
            <NavLink
              to="/"
              className={classNames(
                "btn btn-sm btn-ghost shrink-0 gap-2 text-base-content transition-colors",
                {
                  "bg-base-200 text-primary": selected === "service",
                },
              )}
            >
              <Server className="w-4 h-4" />
              <span className="hidden sm:inline">Services</span>
            </NavLink>

            <NavLink
              to="/proxy"
              className={classNames(
                "btn btn-sm btn-ghost shrink-0 gap-2 text-base-content transition-colors",
                {
                  "bg-base-200 text-primary": selected === "proxy",
                },
              )}
            >
              <Waypoints className="w-4 h-4" />
              <span className="hidden sm:inline">Proxy</span>
            </NavLink>

            <NavLink
              to="/settings"
              className={classNames(
                "btn btn-sm btn-ghost shrink-0 gap-2 text-base-content transition-colors",
                {
                  "bg-base-200 text-primary": selected === "settings",
                },
              )}
            >
              <Settings className="w-4 h-4" />
              <span className="hidden sm:inline">Settings</span>
            </NavLink>

            <NavLink
              to="/operations"
              className={classNames(
                "btn btn-sm btn-ghost shrink-0 gap-2 text-base-content transition-colors",
                {
                  "bg-base-200 text-primary": selected === "operations",
                },
              )}
            >
              <History className="w-4 h-4" />
              <span className="hidden sm:inline">Operations</span>
            </NavLink>
            <NavLink
              to="/status"
              aria-label="System Resources"
              className={classNames(
                "btn btn-sm btn-ghost shrink-0 gap-2 text-base-content transition-colors",
                {
                  "bg-base-200 text-primary": selected === "status",
                },
              )}
            >
              <Activity className="w-4 h-4" aria-hidden="true" />
              <span className="hidden sm:inline">Resources</span>
            </NavLink>
          </nav>
          <div className="dropdown dropdown-end shrink-0">
            <label
              tabIndex={0}
              className="btn btn-sm btn-ghost gap-2 px-2 sm:px-3"
            >
              <User className="w-4 h-4" />
              <span className="hidden sm:inline">Account</span>
            </label>
            <ul
              tabIndex={0}
              className="dropdown-content menu p-2 shadow bg-base-100 rounded-box w-48 mt-2 z-[1]"
            >
              <li>
                <NavLink to="/settings" onClick={closeDropdown}>
                  <Settings className="w-4 h-4" />
                  Settings
                </NavLink>
              </li>
              <li>
                <button onClick={logout}>
                  <LogOut className="w-4 h-4" />
                  Sign out
                </button>
              </li>
            </ul>
          </div>
        </div>
      </header>

      <main
        style={{ height: height - DASHBOARD_LAYOUT_APP_BAR_HEIGHT }}
        className="w-full flex-1 overflow-auto px-2 py-4 sm:px-4 sm:py-6"
      >
        {children || <Outlet />}
      </main>
    </div>
  );
};

import { Link, useLocation } from "react-router";

export function Breadcrumbs() {
  const location = useLocation();
  const segments = location.pathname.split("/").filter(Boolean);

  if (segments.length === 0) return null;

  const crumbs = segments.map((segment, index) => {
    const path = `/${segments.slice(0, index + 1).join("/")}`;
    const label = segment.charAt(0).toUpperCase() + segment.slice(1);
    const isLast = index === segments.length - 1;

    return { path, label, isLast };
  });

  return (
    <nav className="mb-4 flex items-center gap-1 text-sm text-gray-500">
      <Link to="/" className="hover:text-gray-700">
        Home
      </Link>
      {crumbs.map((crumb) => (
        <span key={crumb.path} className="flex items-center gap-1">
          <span>/</span>
          {crumb.isLast ? (
            <span className="font-medium text-gray-900">{crumb.label}</span>
          ) : (
            <Link to={crumb.path} className="hover:text-gray-700">
              {crumb.label}
            </Link>
          )}
        </span>
      ))}
    </nav>
  );
}

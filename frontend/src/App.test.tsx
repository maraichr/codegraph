import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";
import App from "./App";

function renderApp(route = "/") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[route]}>
        <App />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("App", () => {
  it("renders the sidebar with Lattice branding", () => {
    renderApp();
    expect(screen.getByText("Lattice")).toBeInTheDocument();
  });

  it("renders the Projects heading on the home page", () => {
    renderApp("/");
    expect(screen.getByText("Projects")).toBeInTheDocument();
  });
});

import * as React from 'react';
import { Brand, Masthead, MastheadBrand, MastheadMain, Page, SkipToContent } from '@patternfly/react-core';
import logo from '@app/bgimages/WorkloadDriver-Logo.svg';

interface IAppLayout {
  children: React.ReactNode;
}

const AppLayout: React.FunctionComponent<IAppLayout> = ({ children }) => {
  // const [sidebarOpen, setSidebarOpen] = React.useState(true);
  const Header = (
    <Masthead>
      {/* <MastheadToggle>
        <Button variant="plain" onClick={() => setSidebarOpen(!sidebarOpen)} aria-label="Global navigation">
          <BarsIcon />
        </Button>
      </MastheadToggle> */}
      <MastheadMain>
        <MastheadBrand>
          <Brand src={logo} alt="Workload Driver Logo" heights={{ default: '36px' }} />
        </MastheadBrand>
      </MastheadMain>
    </Masthead>
  );

  // const location = useLocation();

  // const renderNavItem = (route: IAppRoute, index: number) => (
  //   <NavItem key={`${route.label}-${index}`} id={`${route.label}-${index}`} isActive={route.path === location.pathname}>
  //     <NavLink exact={route.exact} to={route.path}>
  //       {route.label}
  //     </NavLink>
  //   </NavItem>
  // );

  // const renderNavGroup = (group: IAppRouteGroup, groupIndex: number) => (
  //   <NavExpandable
  //     key={`${group.label}-${groupIndex}`}
  //     id={`${group.label}-${groupIndex}`}
  //     title={group.label}
  //     isActive={group.routes.some((route) => route.path === location.pathname)}
  //   >
  //     {group.routes.map((route, idx) => route.label && renderNavItem(route, idx))}
  //   </NavExpandable>
  // );

  // const Navigation = (
  //   <Nav id="nav-primary-simple" theme="dark">
  //     <NavList id="nav-list-simple">
  //       {routes.map(
  //         (route, idx) => route.label && (!route.routes ? renderNavItem(route, idx) : renderNavGroup(route, idx)),
  //       )}
  //     </NavList>
  //   </Nav>
  // );

  // const Sidebar = (
  //   <PageSidebar theme="dark">
  //     <PageSidebarBody>{Navigation}</PageSidebarBody>
  //   </PageSidebar>
  // );

  const pageId = 'primary-app-container';

  const PageSkipToContent = (
    <SkipToContent
      onClick={(event) => {
        event.preventDefault();
        const primaryContentContainer = document.getElementById(pageId);
        primaryContentContainer && primaryContentContainer.focus();
      }}
      href={`#${pageId}`}
    >
      Skip to Content
    </SkipToContent>
  );
  return (
    <Page mainContainerId={pageId} header={Header} skipToContent={PageSkipToContent}>
      {children}
    </Page>
  );
};

export { AppLayout };

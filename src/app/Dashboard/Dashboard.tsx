import { createRoot } from "react-dom/client";
import "@patternfly/react-core/dist/styles/base.css";

import React from 'react';
import {
  Badge,
  Bullseye,
  Button,
  Card,
  CardHeader,
  CardTitle,
  CardBody,
  Divider,
  Dropdown,
  DropdownItem,
  DropdownList,
  EmptyState,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateFooter,
  EmptyStateVariant,
  EmptyStateActions,
  Gallery,
  Grid,
  GridItem,
  MenuToggle,
  MenuToggleCheckbox,
  OverflowMenu,
  OverflowMenuControl,
  OverflowMenuDropdownItem,
  OverflowMenuItem,
  PageSection,
  PageSectionVariants,
  Pagination,
  TextContent,
  Text,
  Title,
  Toolbar,
  ToolbarItem,
  ToolbarFilter,
  ToolbarContent,
  Select,
  SelectList,
  SelectOption,
  MenuToggleElement
} from '@patternfly/react-core';

import { KernelList } from "src/app/Components/KernelList";
import { KubernetesNodeList } from "src/app/Components/NodeList";
import { KernelSpecList } from "@app/Components/KernelSpecList";

const Dashboard: React.FunctionComponent = () => (
  <PageSection>
    <Title headingLevel="h1" size="lg">Workload Driver: Dashboard</Title>
    <Grid hasGutter>
      <GridItem span={6}>
        <KernelList />
      </GridItem>
      <GridItem span={6}>
        <KubernetesNodeList />
      </GridItem>
      <GridItem span={6}>
        <KernelSpecList />
      </GridItem>
    </Grid>
  </PageSection>
)

export { Dashboard };

import '@patternfly/react-core/dist/styles/base.css';

import React, { useEffect, useRef } from 'react';
import { Grid, GridItem, PageSection } from '@patternfly/react-core';

import { KernelList } from '@app/Components/KernelList';
import { KubernetesNodeList } from '@app/Components/NodeList';
import { KernelSpecList } from '@app/Components/KernelSpecList';
import { KubernetesNode } from '@data/Kubernetes';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@data/Kernel';
import { MigrationModal } from '@app/Components/Modals/MigrationModal';

export interface DashboardProps {
  nodeRefreshInterval: number;
}

const Dashboard: React.FunctionComponent<DashboardProps> = (props: DashboardProps) => {
  const [nodes, setNodes] = React.useState<KubernetesNode[]>([]);
  const [isMigrateModalOpen, setIsMigrateModalOpen] = React.useState(false);
  const [migrateKernel, setMigrateKernel] = React.useState<DistributedJupyterKernel | null>(null);
  const [migrateReplica, setMigrateReplica] = React.useState<JupyterKernelReplica | null>(null);

  const ignoreResponse = useRef(false);
  async function fetchKubernetesNodes() {
    try {
      console.log('Refreshing Kubernetes nodes.');

      // Make a network request to the backend. The server infrastructure handles proxying/routing the request to the correct host.
      // We're specifically targeting the API endpoint I setup called "nodes".
      const response = await fetch('/api/nodes');

      // Get the response, which will be in JSON format, and decode it into an array of KubernetesNode (which is a TypeScript interface that I defined).
      const respNodes: KubernetesNode[] = await response.json();

      if (!ignoreResponse.current) {
        // console.log('Received nodes: ' + JSON.stringify(respNodes));
        setNodes(respNodes);
        console.log('Successfully refreshed Kubernetes nodes.');
      }
    } catch (e) {
      console.error(e);
    }
  }

  // Fetch the kubernetes nodes from the backend (which itself makes a network call to the Kubernetes API).
  useEffect(() => {
    console.log('Refreshing nodes.');
    ignoreResponse.current = false;
    fetchKubernetesNodes();

    // Periodically refresh the Kubernetes nodes every 120,000ms, or when the user clicks the "refresh" button.
    setInterval(() => {
      console.log('Refreshing nodes.');
      ignoreResponse.current = false;
      fetchKubernetesNodes();
      ignoreResponse.current = true;
    }, props.nodeRefreshInterval * 1000);

    return () => {
      ignoreResponse.current = true;
    };
  }, [props.nodeRefreshInterval]);

  async function manuallyRefreshNodes() {
    ignoreResponse.current = false;
    fetchKubernetesNodes();
    ignoreResponse.current = true;
  }

  const onConfirmMigrateReplica = () => {
    setIsMigrateModalOpen(false);
    setMigrateReplica(null);
    setMigrateKernel(null);
  };

  const onCancelMigrateReplica = () => {
    setIsMigrateModalOpen(false);
    setMigrateReplica(null);
    setMigrateKernel(null);
  };

  const openMigrationModal = (kernel: DistributedJupyterKernel, replica: JupyterKernelReplica) => {
    setMigrateReplica(replica);
    setMigrateKernel(kernel);
    setIsMigrateModalOpen(true);
  };

  return (
    <PageSection>
      <Grid hasGutter>
        <GridItem span={6} rowSpan={3}>
          <KernelList openMigrationModal={openMigrationModal} />
        </GridItem>
        <GridItem span={6} rowSpan={4}>
          <KubernetesNodeList
            manuallyRefreshNodes={manuallyRefreshNodes}
            nodes={nodes}
            refreshInterval={120}
            selectable={false}
          />
        </GridItem>
        <GridItem span={6} rowSpan={1}>
          <KernelSpecList />
        </GridItem>
      </Grid>
      <MigrationModal
        nodes={nodes}
        manuallyRefreshNodes={manuallyRefreshNodes}
        isOpen={isMigrateModalOpen}
        onClose={() => onCancelMigrateReplica()}
        onConfirm={() => onConfirmMigrateReplica}
        targetKernel={migrateKernel}
        targetReplica={migrateReplica}
      />
    </PageSection>
  );
};

export { Dashboard };

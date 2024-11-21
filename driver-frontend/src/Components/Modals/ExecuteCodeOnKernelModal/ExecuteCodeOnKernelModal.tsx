import { ExecuteCodeOnKernelPanel } from '@Components/Kernels/ExecuteCodeOnKernelPanel';
import { Modal } from '@patternfly/react-core';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { DistributedJupyterKernel } from '@src/Data';
import React from 'react';

export interface ExecuteCodeOnKernelProps {
    children?: React.ReactNode;
    kernel: DistributedJupyterKernel | null;
    replicaId?: number;
    isOpen: boolean;
    onClose: () => void;
}

export const ExecuteCodeOnKernelModal: React.FunctionComponent<ExecuteCodeOnKernelProps> = (props) => {
    const { authenticated } = React.useContext(AuthorizationContext);

    React.useEffect(() => {
        // Automatically close the modal of we are logged out.
        if (!authenticated) {
            props.onClose();
        }
    }, [props, authenticated]);

    // Reset state, then call user-supplied onClose function.
    const onClose = () => {
        console.log('Closing execute code modal.');
        props.onClose();
    };

    // Returns the title to use for the Modal depending on whether a specific replica was specified as the target or not.
    const getModalTitle = () => {
        if (props.replicaId) {
            return 'Execute Code on Replica ' + props.replicaId + ' of Kernel ' + props.kernel?.kernelId;
        } else {
            return 'Execute Code on Kernel ' + props.kernel?.kernelId;
        }
    };

    return (
        <Modal width="75%" title={getModalTitle()} isOpen={props.isOpen} onClose={props.onClose}>
            <ExecuteCodeOnKernelPanel kernel={props.kernel} replicaId={props.replicaId} onCancel={onClose} />
        </Modal>
    );
};

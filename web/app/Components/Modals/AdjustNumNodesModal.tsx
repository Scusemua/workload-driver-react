import { useNodes } from '@app/Providers';
import {
    Button,
    Form,
    FormGroup,
    FormHelperText,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    Modal,
    ModalVariant,
    TextInput,
} from '@patternfly/react-core';
import { ExclamationCircleIcon, MinusIcon, PlusIcon } from '@patternfly/react-icons';
import React, { FormEvent } from 'react';

export interface AdjustNumNodesModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (value: number, operation: 'set_nodes' | 'add_nodes' | 'remove_nodes') => Promise<void>;
    titleIconVariant?: 'success' | 'danger' | 'warning' | 'info';
}

export const AdjustNumNodesModal: React.FunctionComponent<AdjustNumNodesModalProps> = (props) => {
    type validate = 'success' | 'warning' | 'error' | 'default';

    const { nodes } = useNodes();
    const [targetNumNodes, setTargetNumNodes] = React.useState<string>(nodes.length >= 3 ? `${nodes.length}` : '4');
    const [validated, setValidated] = React.useState<validate>('success');

    const handleTargetNumNodesChanged = (_event: FormEvent<HTMLInputElement>, target: string) => {
        setTargetNumNodes(target);
        if (target === '') {
            setValidated('default');
        } else if (/^\d+$/.test(target)) {
            const targetNum: number = Number.parseInt(target);

            if (targetNum < 3) {
                setValidated('error');
            } else {
                setValidated('success');
            }
        } else {
            setValidated('error');
        }
    };

    const onCloseClicked = () => {
        props.onClose();
    };

    const onConfirmSetNodes = () => {
        if (validated !== 'success') return;

        const target: number = Number.parseInt(targetNumNodes);
        if (target < 3) return;

        props.onConfirm(target, 'set_nodes');
    };

    const onConfirmRemoveNode = () => {
        props.onConfirm(1, 'remove_nodes');
    };

    const onConfirmAddNode = () => {
        props.onConfirm(1, 'add_nodes');
    };

    return (
        <Modal
            variant={ModalVariant.large}
            titleIconVariant={props.titleIconVariant}
            title={'Adjust the Number of Nodes in the Cluster'}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button
                    key="confirm-set-num-nodes"
                    variant="primary"
                    onClick={onConfirmSetNodes}
                    isDisabled={validated !== 'success'}
                >
                    Scale {Number.parseInt(targetNumNodes) > nodes.length ? 'Out' : 'In'} to {targetNumNodes || '?'}{' '}
                    Nodes
                </Button>,
                <Button key="confirm-add-one-node" icon={<PlusIcon />} variant={'primary'} onClick={onConfirmAddNode}>
                    Add 1 Node
                </Button>,
                <Button
                    key="confirm-remove-one-node"
                    icon={<MinusIcon />}
                    variant={'danger'}
                    onClick={onConfirmRemoveNode}
                    isDisabled={nodes.length <= 3}
                >
                    Remove 1 Node
                </Button>,
                <Button key="cancel-adjust-num-nodes" variant="secondary" onClick={onCloseClicked}>
                    Cancel
                </Button>,
            ]}
        >
            <Form>
                <FormGroup label={`Desired number of nodes (current: ${nodes.length})`}>
                    <Grid span={12} hasGutter>
                        <GridItem span={4}>
                            <TextInput
                                id="desired-num-cluster-nodes"
                                aria-label="Desired number of nodes within the cluster"
                                type="number"
                                value={targetNumNodes}
                                onChange={handleTargetNumNodesChanged}
                                validated={validated}
                                placeholder={nodes.length > 0 ? `${nodes.length}` : ''}
                            />
                            {validated !== 'success' && (
                                <FormHelperText>
                                    <HelperText>
                                        <HelperTextItem icon={<ExclamationCircleIcon />} variant={validated}>
                                            {validated === 'error'
                                                ? 'Must be a number ≥ 3'
                                                : 'Please enter your desired number of nodes'}
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
                            )}
                        </GridItem>
                    </Grid>
                </FormGroup>
            </Form>
        </Modal>
    );
};

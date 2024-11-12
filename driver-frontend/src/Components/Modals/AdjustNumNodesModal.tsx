import {
	Button,
	Divider,
	Flex,
	FlexItem,
	Form,
	FormGroup,
	FormHelperText,
	HelperText,
	HelperTextItem,
	TextInput
} from '@patternfly/react-core';
import {
	Modal,
	ModalVariant
} from '@patternfly/react-core/deprecated';
import { ExclamationCircleIcon, MinusIcon, PlusIcon } from '@patternfly/react-icons';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { useNodes } from '@src/Providers';
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
    const [setScaleTargetNumNodes, setSetScaleTargetNumNodes] = React.useState<string>(
        nodes.length >= 3 ? `${nodes.length}` : '4',
    );
    const [addNumNodes, setAddNumNodes] = React.useState<string>('1');
    const [removeNumNodes, setRemoveNumNodes] = React.useState<string>('1');
    const [setNodesValidated, setSetNodesValidated] = React.useState<validate>('success');
    const [addNodesValidated, setAddNodesValidated] = React.useState<validate>('success');
    const [removeNodesValidated, setRemoveNodesValidated] = React.useState<validate>('success');

    const { authenticated } = React.useContext(AuthorizationContext);

    React.useEffect(() => {
        // Automatically close the modal of we are logged out.
        if (!authenticated) {
            props.onClose();
        }
    }, [props, authenticated]);

    const handleTargetNumNodesChanged = (_event: FormEvent<HTMLInputElement>, target: string) => {
        setSetScaleTargetNumNodes(target);
        if (target === '') {
            setSetNodesValidated('default');
        } else if (/^\d+$/.test(target)) {
            const targetNum: number = Number.parseInt(target);

            if (targetNum < 3) {
                setSetNodesValidated('error');
            } else {
                setSetNodesValidated('success');
            }
        } else {
            setSetNodesValidated('error');
        }
    };

    const handleAddNumNodesChanged = (_event: FormEvent<HTMLInputElement>, target: string) => {
        setAddNumNodes(target);
        if (target === '') {
            setAddNodesValidated('default');
        } else if (/^\d+$/.test(target)) {
            const n: number = Number.parseInt(target);

            if (n <= 0) {
                setAddNodesValidated('error');
            } else {
                setAddNodesValidated('success');
            }
        } else {
            setAddNodesValidated('error');
        }
    };

    const handleRemoveNumNodesChanged = (_event: FormEvent<HTMLInputElement>, target: string) => {
        setRemoveNumNodes(target);
        if (target === '') {
            setRemoveNodesValidated('default');
        } else if (/^\d+$/.test(target)) {
            const n: number = Number.parseInt(target);

            if (n <= 0) {
                setRemoveNodesValidated('error');
            } else {
                setRemoveNodesValidated('success');
            }
        } else {
            setRemoveNodesValidated('error');
        }
    };

    const onCloseClicked = () => {
        props.onClose();
    };

    const onConfirmSetNodes = () => {
        if (setNodesValidated !== 'success') return;

        const targetScale: number = Number.parseInt(setScaleTargetNumNodes);
        if (targetScale < 3) return;

        props.onConfirm(targetScale, 'set_nodes').then(() => {});
    };

    const onConfirmRemoveNode = () => {
        if (removeNodesValidated !== 'success') return;

        const n: number = Number.parseInt(removeNumNodes);
        if (n <= 0) return;

        props.onConfirm(n, 'remove_nodes').then(() => {});
    };

    const onConfirmAddNode = () => {
        if (addNodesValidated !== 'success') return;

        const n: number = Number.parseInt(addNumNodes);
        if (n <= 0) return;

        props.onConfirm(n, 'add_nodes').then(() => {});
    };

    return (
        <Modal
            variant={ModalVariant.large}
            titleIconVariant={props.titleIconVariant}
            title={'Adjust the Number of Nodes in the Cluster'}
            isOpen={props.isOpen}
            width={1350}
            onClose={props.onClose}
            actions={[
                <Button key="cancel-adjust-num-nodes" variant="secondary" onClick={onCloseClicked}>
                    Cancel
                </Button>,
            ]}
        >
            <Form>
                <Flex
                    direction={{ default: 'row' }}
                    spaceItems={{ default: 'spaceItemsMd' }}
                    justifyContent={{ default: 'justifyContentCenter' }}
                >
                    <FlexItem>
                        <FormGroup label={`Scale to Target Cluster Size`}>
                            <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                <FlexItem>
                                    <TextInput
                                        id="set-nodes-text-input"
                                        aria-label="Scale the cluster to 'this' many nodes."
                                        type="number"
                                        value={setScaleTargetNumNodes}
                                        onChange={handleTargetNumNodesChanged}
                                        validated={setNodesValidated}
                                        placeholder={nodes.length > 0 ? `${nodes.length}` : ''}
                                    />
                                    {setNodesValidated !== 'success' && (
                                        <FormHelperText>
                                            <HelperText>
                                                <HelperTextItem
                                                    icon={<ExclamationCircleIcon />}
                                                    variant={setNodesValidated}
                                                >
                                                    {setNodesValidated === 'error'
                                                        ? 'Must be a number ≥ 3'
                                                        : 'Please enter your desired number of nodes'}
                                                </HelperTextItem>
                                            </HelperText>
                                        </FormHelperText>
                                    )}
                                </FlexItem>
                                <FlexItem>
                                    <Button
                                        key="confirm-set-num-nodes"
                                        variant="primary"
                                        onClick={onConfirmSetNodes}
                                        isDisabled={setNodesValidated !== 'success'}
                                    >
                                        Scale {Number.parseInt(setScaleTargetNumNodes) > nodes.length ? 'Out' : 'In'} to{' '}
                                        {setScaleTargetNumNodes || '?'} Nodes
                                    </Button>
                                </FlexItem>
                            </Flex>
                        </FormGroup>
                    </FlexItem>
                    <Divider orientation={{ default: 'vertical' }} />
                    <FlexItem>
                        <FormGroup label={`Add Nodes to Cluster`}>
                            <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                <FlexItem>
                                    <TextInput
                                        id="add-nodes-text-input"
                                        aria-label="Add 'this' many nodes to the cluster"
                                        type="number"
                                        value={addNumNodes}
                                        onChange={handleAddNumNodesChanged}
                                        validated={addNodesValidated}
                                        placeholder={nodes.length > 0 ? `${nodes.length}` : ''}
                                    />
                                    {addNodesValidated !== 'success' && (
                                        <FormHelperText>
                                            <HelperText>
                                                <HelperTextItem
                                                    icon={<ExclamationCircleIcon />}
                                                    variant={addNodesValidated}
                                                >
                                                    {addNodesValidated === 'error'
                                                        ? 'Must be a number ≥ 3'
                                                        : 'Please enter your desired number of nodes'}
                                                </HelperTextItem>
                                            </HelperText>
                                        </FormHelperText>
                                    )}
                                </FlexItem>
                                <FlexItem>
                                    <Button
                                        key="confirm-add-nodes-button"
                                        id="confirm-add-nodes-button"
                                        icon={<PlusIcon />}
                                        variant={'primary'}
                                        onClick={onConfirmAddNode}
                                    >
                                        {`Add ${addNumNodes} ${Number.parseInt(addNumNodes) == 1 ? 'Node' : 'Nodes'}`}
                                    </Button>
                                </FlexItem>
                            </Flex>
                        </FormGroup>
                    </FlexItem>
                    <Divider orientation={{ default: 'vertical' }} />
                    <FlexItem>
                        <FormGroup label={'Remove Nodes from Cluster'}>
                            <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                <FlexItem>
                                    <TextInput
                                        id="remove-nodes-text-input"
                                        aria-label="Remove 'this' many nodes from the cluster"
                                        type="number"
                                        value={removeNumNodes}
                                        onChange={handleRemoveNumNodesChanged}
                                        validated={removeNodesValidated}
                                        placeholder={nodes.length > 0 ? `${nodes.length}` : ''}
                                    />
                                    {removeNodesValidated !== 'success' && (
                                        <FormHelperText>
                                            <HelperText>
                                                <HelperTextItem
                                                    icon={<ExclamationCircleIcon />}
                                                    variant={removeNodesValidated}
                                                >
                                                    {removeNodesValidated === 'error'
                                                        ? 'Must be a number ≥ 3'
                                                        : 'Please enter your desired number of nodes'}
                                                </HelperTextItem>
                                            </HelperText>
                                        </FormHelperText>
                                    )}
                                </FlexItem>
                                <FlexItem>
                                    <Button
                                        key="confirm-remove-one-node"
                                        icon={<MinusIcon />}
                                        variant={'danger'}
                                        onClick={onConfirmRemoveNode}
                                        isDisabled={nodes.length <= 3}
                                    >
                                        {`Remove ${removeNumNodes} ${Number.parseInt(removeNumNodes) == 1 ? 'Node' : 'Nodes'}`}
                                    </Button>
                                </FlexItem>
                            </Flex>
                        </FormGroup>
                    </FlexItem>
                    <Divider orientation={{ default: 'vertical' }} />
                </Flex>
            </Form>
        </Modal>
    );
};

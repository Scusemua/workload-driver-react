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
import React from 'react';

export interface AdjustNumNodesModalProps {
    children?: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
    onConfirm: (value: number) => Promise<void>;
    titleIconVariant?: 'success' | 'danger' | 'warning' | 'info';
}

export const AdjustNumNodesModal: React.FunctionComponent<AdjustNumNodesModalProps> = (props) => {
    type validate = 'success' | 'warning' | 'error' | 'default';

    const { nodes } = useNodes();
    const [targetNumNodes, setTargetNumNodes] = React.useState<string>(nodes.length >= 3 ? `${nodes.length}` : '4');
    const [validated, setValidated] = React.useState<validate>('success');

    const handleTargetNumNodesChanged = (_event, target: string) => {
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

    const onConfirmClicked = () => {
        if (validated !== 'success') return;

        const target: number = Number.parseInt(targetNumNodes);
        if (target < 3) return;

        props.onConfirm(target);
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
                    key="confirm-adjust-num-nodes"
                    variant="primary"
                    onClick={onConfirmClicked}
                    isDisabled={validated !== 'success'}
                >
                    Confirm
                </Button>,
                <Button key="cancel-adjust-num-nodes" variant="link" onClick={onCloseClicked}>
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
                        {/*<GridItem span={2}>*/}
                        {/*    <Button icon={<PlusIcon />} variant={'secondary'}>*/}
                        {/*        Add 1 Node*/}
                        {/*    </Button>*/}
                        {/*</GridItem>*/}
                        {/*<GridItem span={2}>*/}
                        {/*    <Button icon={<MinusIcon />} variant={'secondary'} isDanger isDisabled={nodes.length <= 3}>*/}
                        {/*        Remove 1 Node*/}
                        {/*    </Button>*/}
                        {/*</GridItem>*/}
                    </Grid>
                </FormGroup>
            </Form>
        </Modal>
    );
};

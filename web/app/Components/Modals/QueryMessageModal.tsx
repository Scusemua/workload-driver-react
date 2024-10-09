import { GetToastContentWithHeaderAndBody } from '@app/utils/toast_utils';
import { QueryMessageResponse } from '@data/Message';
import {
    Button,
    Card,
    CardBody,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    FormHelperText,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    Label,
    Modal,
    ModalVariant,
    TextInput,
} from '@patternfly/react-core';
import { CheckCircleIcon, TimesCircleIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { global_BackgroundColor_150 } from '@patternfly/react-tokens';
import React from 'react';
import toast from 'react-hot-toast';

interface QueryMessageModalProps {
    isOpen: boolean;
    onClose: () => void;
}

export const QueryMessageModal: React.FunctionComponent<QueryMessageModalProps> = (props: QueryMessageModalProps) => {
    const [jupyterMsgId, setJupyterMsgId] = React.useState<string>('');
    const [jupyterMsgType, setJupyterMsgType] = React.useState<string>('');
    const [jupyterKernelId, setJupyterKernelId] = React.useState<string>('');
    const [queryResults, setQueryResults] = React.useState<QueryMessageResponse[]>([]);

    const onSubmitClicked = () => {
        const req: RequestInit = {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                messageId: jupyterMsgId,
                messageType: jupyterMsgType,
                kernelId: jupyterKernelId,
            }),
        };

        let loadingText: string = `Querying status of Jupyter ZMQ message "${jupyterMsgId}"`;
        if (jupyterMsgType != '') {
            loadingText += ` of type ${jupyterMsgType}`;
        }
        if (jupyterKernelId != '') {
            loadingText += ` targeting kernel ${jupyterKernelId}`;
        }
        loadingText += ' now...';

        const toastId: string = toast.loading(loadingText, { style: { maxWidth: 750 } });

        fetch('api/query-message', req)
            .catch((err: Error) => {
                console.log(`QueryMessage failed: ${JSON.stringify(err)}`);
                toast.custom(
                    GetToastContentWithHeaderAndBody(
                        `Failed to query status of Jupyter ZMQ message "${jupyterMsgId}"`,
                        `Reason: ${err.message}`,
                      'danger',
                      () => {toast.dismiss(toastId)}
                    ),
                    { id: toastId, style: { maxWidth: 750 } },
                );
            })
            .then(async (resp: Response | void) => {
                if (resp?.status == 200) {
                    const queryResult: QueryMessageResponse = await resp.json().catch(()=>{console.error("AAA");});
                    toast.custom(
                        GetToastContentWithHeaderAndBody(
                            `Successfully queried status of Jupyter ZMQ message "${jupyterMsgId}"`,
                            JSON.stringify(queryResult),
                          'success',
                          () => {toast.dismiss(toastId)}
                        ),
                        { id: toastId, style: { maxWidth: 750 } },
                    );

                    setQueryResults((prevResults) => {
                        return [...prevResults, queryResult];
                    });
                } else {
                    const responseContent = await resp?.json();

                    // HTTP 400 here just means that the Gateway didn't have any such request whatsoever.
                    if (resp?.status == 400) {
                      // We'll add an entry for this query, since we know the Gateway simply didn't have
                      // the requested request in its request log.
                      const queryResult: QueryMessageResponse = {
                        messageId: jupyterMsgId,
                        kernelId: jupyterKernelId,
                        messageType: jupyterMsgType,
                        gatewayReceivedRequest: -1,
                        gatewayForwardedRequest: -1,
                        gatewayReceivedReply: -1,
                        gatewayForwardedReply: -1,
                      }

                      setQueryResults((prevResults) => {
                        return [...prevResults, queryResult];
                      });

                      toast.custom(
                        GetToastContentWithHeaderAndBody(
                          `Request Not Found`,
                          `${responseContent["message"]}`,
                          'danger',
                          () => {toast.dismiss(toastId)}
                        ),
                        { id: toastId, style: { maxWidth: 750 } },
                      );
                    } else {
                      // Unknown/unexpected error. Display a warning.
                      toast.custom(
                        GetToastContentWithHeaderAndBody(
                          `Failed to query status of Jupyter ZMQ message "${jupyterMsgId}"`,
                          `HTTP ${resp?.status} ${resp?.statusText}: ${responseContent["message"]}`,
                          'danger',
                          () => {toast.dismiss(toastId)}
                        ),
                        { id: toastId, style: { maxWidth: 750 } },
                      );
                    }
                }
            });
    };

    const queryForm = (
        <Form>
            <Grid span={12} hasGutter>
                <GridItem span={4}>
                    <FormGroup label={'Jupyter Message ID'} isRequired>
                        <TextInput
                            validated={jupyterMsgId.length > 0 ? 'success' : 'warning'}
                            isRequired
                            type="text"
                            id="query-message-jupyter-msg-id-field"
                            name="query-message-jupyter-msg-id-field"
                            aria-label="query-message-jupyter-msg-id-field"
                            value={jupyterMsgId}
                            onChange={(_event, msg_id: string) => setJupyterMsgId(msg_id)}
                        />
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem>
                                    Specify the ID of the Jupyter message as it appears in the Jupyter message&apos;s
                                    header.
                                </HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    </FormGroup>
                </GridItem>
                <GridItem span={4}>
                    <FormGroup label={'Jupyter Kernel ID'} isRequired>
                        <TextInput
                            isRequired
                            type="text"
                            id="query-message-jupyter-kernel-id-field"
                            name="query-message-jupyter-kernel-id-field"
                            aria-label="query-message-jupyter-kernel-id-field"
                            value={jupyterKernelId}
                            onChange={(_event, msg_id: string) => setJupyterKernelId(msg_id)}
                        />
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem>Specify the ID of the target Jupyter kernel.</HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    </FormGroup>
                </GridItem>
                <GridItem span={4}>
                    <FormGroup label={'Jupyter Message ID'} isRequired>
                        <TextInput
                            isRequired
                            type="text"
                            id="query-message-jupyter-msg-type-field"
                            name="query-message-jupyter-msg-type-field"
                            aria-label="query-message-jupyter-msg-type-field"
                            value={jupyterMsgType}
                            onChange={(_event, msg_id: string) => setJupyterMsgType(msg_id)}
                        />
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem>
                                    Specify the type of the Jupyter message as it appears in the Jupyter message&apos;s
                                    header.
                                </HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    </FormGroup>
                </GridItem>
            </Grid>
        </Form>
    );

    const columnNames = {
        seen: 'Seen',
        messageId: 'MsgID',
        messageType: 'MsgType',
        kernelId: 'KernelId',
        gatewayReceivedRequest: 'Gateway Recv Req',
        gatewayForwardedRequest: 'Gateway Sent Req',
        gatewayReceivedReply: 'Gateway Recv Reply',
        gatewayForwardedReply: 'Gateway Sent Reply',
    };

    const getLabel = (ts: number) => {
        if (ts > 0) {
            return <Label icon={<CheckCircleIcon />} color={'green'} />;
        } else {
            return <Label icon={<TimesCircleIcon />} color={'red'} />;
        }
    };

    const getMsgTypeRow = (queryResult: QueryMessageResponse) => {
        if (!queryResult.messageType || queryResult.messageType.length == 0) {
            return '-';
        }

        return queryResult.messageType;
    };

    const getKernelIdRow = (queryResult: QueryMessageResponse) => {
        if (!queryResult.kernelId || queryResult.kernelId.length == 0) {
            return '-';
        }

        return queryResult.kernelId;
    };

    const queryResultVisualization = (
        <Card isRounded isCompact>
            <CardBody>
                <Table variant={'compact'} aria-label={'Message query result table'}>
                    <Thead noWrap>
                        <Tr>
                            <Th>{columnNames.messageId}</Th>
                            <Th>{columnNames.messageType}</Th>
                            <Th>{columnNames.kernelId}</Th>
                            <Th>{columnNames.gatewayReceivedRequest}</Th>
                            <Th>{columnNames.gatewayForwardedRequest}</Th>
                            <Th>{columnNames.gatewayReceivedReply}</Th>
                            <Th>{columnNames.gatewayForwardedReply}</Th>
                            <Th screenReaderText={'Dismiss button'} />
                        </Tr>
                    </Thead>
                    <Tbody>
                        {queryResults.map((queryResult: QueryMessageResponse, rowIndex: number) => {
                            const isOddRow = (rowIndex + 1) % 2;
                            const customStyle = {
                                backgroundColor: global_BackgroundColor_150.var,
                            };

                            return (
                                <Tr
                                    key={`${rowIndex}-${queryResult.messageId}`}
                                    className={isOddRow ? 'odd-row-class' : 'even-row-class'}
                                    style={isOddRow ? customStyle : {}}
                                >
                                    <Td dataLabel={columnNames.messageId}>{queryResult.messageId}</Td>
                                    <Td dataLabel={columnNames.messageType}>{getMsgTypeRow(queryResult)}</Td>
                                    <Td dataLabel={columnNames.kernelId}>{getKernelIdRow(queryResult)}</Td>
                                    <Td dataLabel={columnNames.gatewayReceivedRequest}>
                                        {getLabel(queryResult.gatewayReceivedRequest)}
                                    </Td>
                                    <Td dataLabel={columnNames.gatewayForwardedRequest}>
                                        {getLabel(queryResult.gatewayForwardedRequest)}
                                    </Td>
                                    <Td dataLabel={columnNames.gatewayReceivedReply}>
                                        {getLabel(queryResult.gatewayReceivedReply)}
                                    </Td>
                                    <Td dataLabel={columnNames.gatewayForwardedReply}>
                                        {getLabel(queryResult.gatewayForwardedReply)}
                                    </Td>
                                    <Td modifier="fitContent">
                                        <Button
                                            key={`dismiss-query-result-${rowIndex}-button`}
                                            variant="link"
                                            onClick={() => {
                                                setQueryResults((prevResults) => {
                                                    // if (prevResults.length == 1) {
                                                    //     return [];
                                                    // }

                                                    return prevResults.filter(
                                                        (_: QueryMessageResponse, index: number) => index != rowIndex,
                                                    );

                                                    // return [
                                                    //     ...prevResults.slice(0, rowIndex),
                                                    //     ...prevResults.slice[rowIndex + 1],
                                                    // ];
                                                });
                                            }}
                                        >
                                            Dismiss
                                        </Button>
                                    </Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                </Table>
            </CardBody>
        </Card>
    );

    return (
        <Modal
            variant={ModalVariant.large}
            titleIconVariant={'info'}
            title={'Query Status of Jupyter ZMQ Message'}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button
                    key="submit-query-message-modal-button"
                    variant="primary"
                    onClick={onSubmitClicked}
                    isDisabled={jupyterMsgId.length == 0}
                >
                    Submit
                </Button>,
                <Button key="dismiss-query-message-modal-button" variant="primary" onClick={props.onClose}>
                    Dismiss
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                <FlexItem>{queryForm}</FlexItem>
                <FlexItem hidden={queryResults.length == 0}>{queryResultVisualization}</FlexItem>
            </Flex>
        </Modal>
    );
};

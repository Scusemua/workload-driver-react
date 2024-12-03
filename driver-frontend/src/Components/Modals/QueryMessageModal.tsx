import { QueryMessageResponse, RequestTrace } from '@Data/Message';
import {
    Badge,
    Button,
    Card,
    CardBody,
    CardHeader,
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
    MenuToggle,
    MenuToggleElement,
    Modal,
    ModalVariant,
    Pagination,
    PaginationVariant,
    SearchInput,
    Select,
    SelectList,
    SelectOption,
    Skeleton,
    TextInput,
    Toolbar,
    ToolbarContent,
    ToolbarFilter,
    ToolbarGroup,
    ToolbarItem,
    ToolbarToggleGroup,
} from '@patternfly/react-core';
import { CheckCircleIcon, FilterIcon, SearchIcon, TimesCircleIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { global_BackgroundColor_150 } from '@patternfly/react-tokens';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { GetPathForFetch } from '@src/Utils/path_utils';
import { GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import { RoundToThreeDecimalPlaces } from '@Utils/utils';
import { MAX_SAFE_INTEGER } from 'lib0/number';
import React from 'react';
import toast from 'react-hot-toast';
import { v4 as uuidv4 } from 'uuid';

interface QueryMessageModalProps {
    isOpen: boolean;
    onClose: () => void;
}

interface Filters {
    messageType: string[];
}

export const QueryMessageModal: React.FunctionComponent<QueryMessageModalProps> = (props: QueryMessageModalProps) => {
    const [kernelIdFilter, setKernelIdFilter] = React.useState<string>('');
    const [messageIdFilter, setMessageIdFilter] = React.useState<string>('');
    const [messageTypeFilterIsExpanded, setMessageTypeFilterIsExpanded] = React.useState<boolean>(false);
    const [filters, setFilters] = React.useState<Filters>({
        messageType: [],
    });

    const [queryIsActive, setQueryIsActive] = React.useState<boolean>(false);

    const [jupyterMsgId, setJupyterMsgId] = React.useState<string>('');
    const [jupyterMsgType, setJupyterMsgType] = React.useState<string>('');
    const [jupyterKernelId, setJupyterKernelId] = React.useState<string>('');
    const [requestTraces, setRequestTraces] = React.useState<Map<string, RequestTrace>>(
        new Map<string, RequestTrace>(),
    );
    const [possibleMessageTypes, setPossibleMessageTypes] = React.useState<Set<string>>(new Set<string>());

    const [paginationPage, setPaginationPage] = React.useState<number>(1);
    const [resultsPerPage, setResultsPerPage] = React.useState<number>(5);

    const { authenticated, setAuthenticated } = React.useContext(AuthorizationContext);

    const onSetPage = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPage: number) => {
        setPaginationPage(newPage);
    };

    const onPerPageSelect = (
        _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
        newPerPage: number,
        newPage: number,
    ) => {
        setResultsPerPage(newPerPage);
        setPaginationPage(newPage);
    };

    React.useEffect(() => {
        const messageTypes: Set<string> = new Set<string>();
        requestTraces.forEach((trace: RequestTrace) => {
            messageTypes.add(trace.messageType);
        });

        setPossibleMessageTypes(messageTypes);
    }, [requestTraces]);

    React.useEffect(() => {
        // If we are logged out, then close the modal.
        if (!authenticated) {
            props.onClose();
        }
    }, [authenticated, props]);

    const onSubmitClicked = () => {
        if (!authenticated) {
            return;
        }

        let targetMsgId: string = jupyterMsgId;
        if (targetMsgId.length == 0) {
            targetMsgId = '*';
        }

        const req: RequestInit = {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                Authorization: 'Bearer ' + localStorage.getItem('token'),
            },
            body: JSON.stringify({
                messageId: targetMsgId,
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

        const getToastBody = (queryMessageResponse: QueryMessageResponse, latencyMs: number): string => {
            if (targetMsgId === '*') {
                if (queryMessageResponse.requestTraces) {
                    return `Server returned ${queryMessageResponse.requestTraces.length} request trace(s) in ${latencyMs} ms.`;
                } else {
                    return `Server returned 0 request trace(s)`;
                }
            } else {
                return JSON.stringify(queryMessageResponse);
            }
        };

        const startTime: number = performance.now();
        setQueryIsActive(true);
        // Whatever you want to do after the wait
        fetch(GetPathForFetch('api/query-message'), req)
            .catch((err: Error) => {
                setQueryIsActive(false);
                console.log(`QueryMessage failed: ${JSON.stringify(err)}`);
                toast.custom(
                    GetToastContentWithHeaderAndBody(
                        `Failed to query status of Jupyter ZMQ message "${jupyterMsgId}"`,
                        `Reason: ${err.message}`,
                        'danger',
                        () => {
                            toast.dismiss(toastId);
                        },
                    ),
                    { id: toastId, style: { maxWidth: 750 } },
                );
            })
            .then(async (resp: Response | void) => {
                if (resp?.status == 200) {
                    const queryMessageResponse: QueryMessageResponse = await resp.json();
                    setQueryIsActive(false);
                    const latencyMs: number = RoundToThreeDecimalPlaces(performance.now() - startTime);
                    toast.custom(
                        GetToastContentWithHeaderAndBody(
                            `Successfully queried status of Jupyter ZMQ message "${jupyterMsgId}" (${latencyMs} ms)`,
                            getToastBody(queryMessageResponse, latencyMs),
                            'success',
                            () => {
                                toast.dismiss(toastId);
                            },
                        ),
                        { id: toastId, style: { maxWidth: 750 } },
                    );

                    if (queryMessageResponse.requestTraces && queryMessageResponse.requestTraces.length > 0) {
                        setRequestTraces((prevResults: Map<string, RequestTrace>) => {
                            const nextResults: Map<string, RequestTrace> = new Map<string, RequestTrace>(prevResults);

                            queryMessageResponse.requestTraces.forEach((val: RequestTrace) => {
                                nextResults.set(getRequestTraceKey(val), val);
                            });

                            return nextResults;
                        });
                    }
                } else {
                    const responseContent = await resp?.json();
                    setQueryIsActive(false);

                    // HTTP 400 here just means that the Gateway didn't have any such request whatsoever.
                    if (resp?.status == 400) {
                        if (jupyterMsgId === '*') {
                            toast.custom(
                                GetToastContentWithHeaderAndBody(
                                    `RequestLog is Empty`,
                                    `There are no requests in the Cluster Gateway's RequestLog.`,
                                    'warning',
                                    () => {
                                        toast.dismiss(toastId);
                                    },
                                ),
                                { id: toastId, style: { maxWidth: 750 } },
                            );

                            return;
                        }

                        // We'll add an entry for this query, since we know the Gateway simply didn't have
                        // the requested request in its request log.
                        const requestTrace: RequestTrace = {
                            messageId: jupyterMsgId,
                            kernelId: jupyterKernelId,
                            messageType: jupyterMsgType,
                            replicaId: -1,
                            requestReceivedByGateway: -1,
                            requestSentByGateway: -1,
                            requestReceivedByLocalDaemon: -1,
                            requestSentByLocalDaemon: -1,
                            requestReceivedByKernelReplica: -1,
                            replySentByKernelReplica: -1,
                            replyReceivedByLocalDaemon: -1,
                            replySentByLocalDaemon: -1,
                            replyReceivedByGateway: -1,
                            replySentByGateway: -1,
                            e2eLatencyMilliseconds: -1,
                            cudaInitMicroseconds: -1,
                            downloadDependencyMicroseconds: -1,
                            downloadModelAndTrainingDataMicroseconds: -1,
                            uploadModelAndTrainingDataMicroseconds: -1,
                            executionTimeMicroseconds: -1,
                            executionStartUnixMillis: -1,
                            executionEndUnixMillis: -1,
                            replayTimeMicroseconds: -1,
                            copyFromCpuToGpuMicroseconds: -1,
                            copyFromGpuToCpuMicroseconds: -1,
                            leaderElectionTimeMicroseconds: -1,
                            electionCreationTime: -1,
                            electionProposalPhaseStartTime: -1,
                            electionExecutionPhaseStartTime: -1,
                            electionEndTime: -1,
                            requestTraceUuid: uuidv4(),
                        };

                        const traceKey: string = getRequestTraceKey(requestTrace);

                        setRequestTraces((prevResults) => {
                            return new Map<string, RequestTrace>(prevResults).set(traceKey, requestTrace);
                        });

                        toast.custom(
                            GetToastContentWithHeaderAndBody(
                                `Request Not Found`,
                                `${responseContent['message']}`,
                                'danger',
                                () => {
                                    toast.dismiss(toastId);
                                },
                            ),
                            { id: toastId, style: { maxWidth: 750 } },
                        );
                    } else if (resp?.status == 401) {
                        setAuthenticated(false);
                    } else {
                        // Unknown/unexpected error. Display a warning.
                        toast.custom(
                            GetToastContentWithHeaderAndBody(
                                `Failed to query status of Jupyter ZMQ message "${jupyterMsgId}"`,
                                `HTTP ${resp?.status} ${resp?.statusText}: ${responseContent['message']}`,
                                'danger',
                                () => {
                                    toast.dismiss(toastId);
                                },
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
                            isRequired
                            type="text"
                            id="query-message-jupyter-msg-id-field"
                            name="query-message-jupyter-msg-id-field"
                            aria-label="query-message-jupyter-msg-id-field"
                            placeholder={'* (i.e., query for all messages)'}
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
        messageId: 'Message ID',
        messageType: 'Message Type',
        kernelId: 'Kernel ID',
        replicaId: 'Replica ID',
        requestReceivedByGateway: 'CG Recv Req',
        requestSentByGateway: 'CG Sent Req',
        replyReceivedByGateway: 'CG Recv Reply',
        replySentByGateway: 'CG Sent Reply',
    };

    const getLabel = (ts: number) => {
        if (ts > 0) {
            return <Label icon={<CheckCircleIcon />} color={'green'} />;
        } else {
            return <Label icon={<TimesCircleIcon />} color={'red'} />;
        }
    };

    const getMsgTypeRow = (requestTrace: RequestTrace) => {
        if (!requestTrace.messageType || requestTrace.messageType.length == 0) {
            return '-';
        }

        return requestTrace.messageType;
    };

    const getReplicaIdRow = (requestTrace: RequestTrace) => {
        if (!requestTrace.replicaId || requestTrace.replicaId == -1) {
            return '-';
        }

        return requestTrace.replicaId;
    };

    const getKernelIdRow = (requestTrace: RequestTrace) => {
        if (!requestTrace.kernelId || requestTrace.kernelId.length == 0) {
            return '-';
        }

        return requestTrace.kernelId;
    };

    const getRequestTraceKey = (requestTrace: RequestTrace): string => {
        return requestTrace.messageId + '-' + requestTrace.replicaId.toString();
    };

    const onSelect = (
        type: string,
        event: React.MouseEvent<Element, MouseEvent> | undefined,
        selection: string | number | undefined,
    ) => {
        const checked = (event?.target as HTMLInputElement).checked;
        setFilters((prev) => {
            console.log(`Previous filters: ${JSON.stringify(prev)}`);
            const prevSelections = prev[type] || [];
            console.log(`Previous selections: ${JSON.stringify(prevSelections)}`);
            const next = {
                ...prev,
                [type]: checked
                    ? [...prevSelections, selection]
                    : prevSelections.filter(
                          (value: string | number | undefined) => (value as string) !== (selection as string),
                      ),
            };
            console.log(`Next/new filters: ${JSON.stringify(next)}`);
            return next;
        });
    };

    const onMessageTypeFilterSelected = (
        event?: React.MouseEvent<Element, MouseEvent> | undefined,
        value?: string | number | undefined,
    ) => {
        onSelect('messageType', event, value);
    };

    const onDeleteGroup = (type: string) => {
        if (type === 'messageType') {
            setFilters({ messageType: [] });
        }
    };

    const onDeleteFilter = (type: string, id: string) => {
        if (type === 'messageType') {
            setFilters({ messageType: filters.messageType.filter((fil: string) => fil !== id) });
        } else {
            setFilters({ messageType: [] });
        }
    };

    const filterByKernelIdSearchBar = (
        <ToolbarItem variant="search-filter">
            <SearchInput
                aria-label="Filter query results by kernel ID"
                placeholder={'Filter by kernel ID'}
                onChange={(_event, value) => setKernelIdFilter(value)}
                value={kernelIdFilter}
                onClear={() => {
                    setKernelIdFilter('');
                }}
            />
        </ToolbarItem>
    );

    const filterByMessageIdSearchBar = (
        <ToolbarItem variant="search-filter">
            <SearchInput
                aria-label="Filter query results by message ID"
                placeholder={'Filter by message ID'}
                onChange={(_event, value) => setMessageIdFilter(value)}
                value={messageIdFilter}
                onClear={() => {
                    setMessageIdFilter('');
                }}
            />
        </ToolbarItem>
    );

    const filterByMessageTypeSelection = (
        <ToolbarGroup variant="filter-group">
            <ToolbarFilter
                chips={filters.messageType}
                deleteChip={(category, chip) => onDeleteFilter(category as string, chip as string)}
                deleteChipGroup={(category) => onDeleteGroup(category as string)}
                categoryName={'Message Type'}
            >
                <Select
                    role={'menu'}
                    onSelect={onMessageTypeFilterSelected}
                    onOpenChange={(isOpen) => setMessageTypeFilterIsExpanded(isOpen)}
                    selected={filters.messageType}
                    isOpen={messageTypeFilterIsExpanded}
                    toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                        <MenuToggle
                            ref={toggleRef}
                            onClick={() => setMessageTypeFilterIsExpanded((expanded: boolean) => !expanded)}
                            isExpanded={messageTypeFilterIsExpanded}
                            icon={<SearchIcon />}
                            badge={filters.messageType.length > 0 && <Badge isRead>{filters.messageType.length}</Badge>}
                            style={
                                {
                                    width: '250px',
                                } as React.CSSProperties
                            }
                        >
                            Message Type
                        </MenuToggle>
                    )}
                >
                    <SelectList>
                        {Array.from(possibleMessageTypes).map((msgType: string) => (
                            <SelectOption
                                key={msgType}
                                value={msgType}
                                hasCheckbox
                                isSelected={filters.messageType.includes(msgType)}
                            >
                                {msgType}
                            </SelectOption>
                        ))}
                    </SelectList>
                </Select>
            </ToolbarFilter>
        </ToolbarGroup>
    );

    const queryTableActions = (
        <Toolbar>
            <ToolbarContent>
                <ToolbarToggleGroup toggleIcon={<FilterIcon />} breakpoint="xl">
                    {filterByKernelIdSearchBar}
                    {filterByMessageIdSearchBar}
                    {filterByMessageTypeSelection}
                </ToolbarToggleGroup>
            </ToolbarContent>
        </Toolbar>
    );

    const onFilter = (value: [string, RequestTrace]) => {
        const requestTrace: RequestTrace = value[1];

        // Search name with search value
        let kernelIdFilterInput: RegExp;
        try {
            kernelIdFilterInput = new RegExp(kernelIdFilter, 'i');
            // eslint-disable-next-line @typescript-eslint/no-unused-vars
        } catch (err) {
            kernelIdFilterInput = new RegExp(kernelIdFilter.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i');
        }
        const matchesKernelId = requestTrace.kernelId.search(kernelIdFilterInput) >= 0;

        // Search name with search value
        let messageIdFilterInput: RegExp;
        try {
            messageIdFilterInput = new RegExp(messageIdFilter, 'i');
            // eslint-disable-next-line @typescript-eslint/no-unused-vars
        } catch (err) {
            messageIdFilterInput = new RegExp(messageIdFilter.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i');
        }
        const matchesMessageId = requestTrace.messageId.search(messageIdFilterInput) >= 0;

        // Search status with status selection
        let matchesStatusValue = false;
        filters.messageType.forEach(function (selectedMessageType: string) {
            const match = requestTrace.messageType.toLowerCase() === selectedMessageType.toLowerCase();
            matchesStatusValue = matchesStatusValue || match;
        });

        return (
            (kernelIdFilter === '' || matchesKernelId) &&
            (messageIdFilter === '' || matchesMessageId) &&
            (filters.messageType.length === 0 || matchesStatusValue)
        );
    };

    const filteredTraces = Array.from(requestTraces).filter(onFilter);
    const paginatedTraces = filteredTraces.slice(
        resultsPerPage * (paginationPage - 1),
        resultsPerPage * (paginationPage - 1) + resultsPerPage,
    );

    const getSkeletonRow = (rowIndex: number) => {
        const isOddRow = (rowIndex + 1) % 2;
        const customStyle = {
            backgroundColor: global_BackgroundColor_150.var,
        };

        return (
            <Tr
                key={`${rowIndex}-skeleton`}
                className={isOddRow ? 'odd-row-class' : 'even-row-class'}
                style={isOddRow ? customStyle : {}}
            >
                <Td dataLabel={columnNames.messageId}>
                    <Skeleton />
                </Td>
                <Td dataLabel={columnNames.messageType}>
                    <Skeleton />
                </Td>
                <Td dataLabel={columnNames.kernelId}>
                    <Skeleton />
                </Td>
                <Td dataLabel={columnNames.replicaId}>
                    <Skeleton />
                </Td>
                <Td dataLabel={columnNames.requestReceivedByGateway}>
                    <Skeleton />
                </Td>
                <Td dataLabel={columnNames.requestSentByGateway}>
                    <Skeleton />
                </Td>
                <Td dataLabel={columnNames.replyReceivedByGateway}>
                    <Skeleton />
                </Td>
                <Td dataLabel={columnNames.replySentByGateway}>
                    <Skeleton />
                </Td>
                <Td modifier="fitContent">
                    <Skeleton />
                </Td>
            </Tr>
        );
    };

    const tableBodyDefinitionDuringFirstQuery = (
        <Tbody>
            {getSkeletonRow(0)}
            {getSkeletonRow(1)}
            {getSkeletonRow(2)}
            {getSkeletonRow(3)}
            {getSkeletonRow(4)}
        </Tbody>
    );

    const tableHeaderDefinition = (
        <Thead noWrap>
            <Tr>
                <Th>{columnNames.messageId}</Th>
                <Th>{columnNames.messageType}</Th>
                <Th>{columnNames.kernelId}</Th>
                <Th>{columnNames.replicaId}</Th>
                <Th>{columnNames.requestReceivedByGateway}</Th>
                <Th>{columnNames.requestSentByGateway}</Th>
                <Th>{columnNames.replyReceivedByGateway}</Th>
                <Th>{columnNames.replySentByGateway}</Th>
                <Th screenReaderText={'Dismiss button'} />
            </Tr>
        </Thead>
    );

    const tableBodyDefinition = (
        <Tbody>
            {paginatedTraces.map(([traceKey, requestTrace], rowIndex: number) => {
                const isOddRow = (rowIndex + 1) % 2;
                const customStyle = {
                    backgroundColor: global_BackgroundColor_150.var,
                };

                return (
                    <Tr
                        key={`${rowIndex}-${requestTrace.messageId}`}
                        className={isOddRow ? 'odd-row-class' : 'even-row-class'}
                        style={isOddRow ? customStyle : {}}
                    >
                        <Td dataLabel={columnNames.messageId}>{requestTrace.messageId}</Td>
                        <Td dataLabel={columnNames.messageType}>{getMsgTypeRow(requestTrace)}</Td>
                        <Td dataLabel={columnNames.kernelId}>{getKernelIdRow(requestTrace)}</Td>
                        <Td dataLabel={columnNames.replicaId}>{getReplicaIdRow(requestTrace)}</Td>
                        <Td dataLabel={columnNames.requestReceivedByGateway}>
                            {getLabel(requestTrace.requestReceivedByGateway)}
                        </Td>
                        <Td dataLabel={columnNames.requestSentByGateway}>
                            {getLabel(requestTrace.requestSentByGateway)}
                        </Td>
                        <Td dataLabel={columnNames.replyReceivedByGateway}>
                            {getLabel(requestTrace.replyReceivedByGateway)}
                        </Td>
                        <Td dataLabel={columnNames.replySentByGateway}>{getLabel(requestTrace.replySentByGateway)}</Td>
                        <Td modifier="fitContent">
                            <Button
                                key={`dismiss-query-result-${rowIndex}-button`}
                                variant="link"
                                onClick={() => {
                                    setRequestTraces((prevResults) => {
                                        const nextResults: Map<string, RequestTrace> = new Map<string, RequestTrace>(
                                            prevResults,
                                        );
                                        nextResults.delete(traceKey);
                                        return nextResults;
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
    );

    const queryResultTable = (
        <Card isRounded isCompact>
            <CardHeader>{queryTableActions}</CardHeader>
            <CardBody>
                <Table variant={'compact'} aria-label={'Message query result table'}>
                    {tableHeaderDefinition}
                    {requestTraces.size == 0 && queryIsActive && tableBodyDefinitionDuringFirstQuery}
                    {requestTraces.size > 0 && tableBodyDefinition}
                </Table>
                <Pagination
                    hidden={filteredTraces.length <= 0}
                    isDisabled={filteredTraces.length <= 0}
                    itemCount={filteredTraces.length > 0 ? filteredTraces.length : 0}
                    widgetId="query-messages-list-pagination"
                    perPage={resultsPerPage}
                    page={paginationPage}
                    variant={PaginationVariant.bottom}
                    perPageOptions={[
                        {
                            title: '3',
                            value: 3,
                        },
                        {
                            title: '5',
                            value: 5,
                        },
                        {
                            title: '10',
                            value: 10,
                        },
                        {
                            title: '15',
                            value: 15,
                        },

                        {
                            title: '∞',
                            value: MAX_SAFE_INTEGER,
                        },
                    ]}
                    onSetPage={onSetPage}
                    onPerPageSelect={onPerPageSelect}
                />
            </CardBody>
        </Card>
    );

    return (
        <Modal
            variant={ModalVariant.large}
            titleIconVariant={'info'}
            maxWidth={1280}
            width={1280}
            title={'Query Status of Jupyter ZMQ Message'}
            isOpen={props.isOpen}
            onClose={props.onClose}
            actions={[
                <Button
                    key="submit-query-message-modal-button"
                    variant="primary"
                    onClick={onSubmitClicked}
                    isDisabled={!authenticated}
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
                <FlexItem hidden={requestTraces.size == 0 && !queryIsActive}>{queryResultTable}</FlexItem>
            </Flex>
        </Modal>
    );
};

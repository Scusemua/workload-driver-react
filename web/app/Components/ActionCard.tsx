import React, { useEffect, useRef } from 'react';
import {
    Badge,
    Button,
    Card,
    CardBody,
    CardExpandableContent,
    CardHeader,
    CardTitle,
    DataList,
    DataListAction,
    DataListCell,
    DataListCheck,
    DataListContent,
    DataListItem,
    DataListItemCells,
    DataListItemRow,
    DataListToggle,
    Flex,
    FlexItem,
    InputGroup,
    InputGroupItem,
    Menu,
    MenuContent,
    MenuItem,
    MenuList,
    MenuToggle,
    Popper,
    SearchInput,
    Stack,
    StackItem,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarFilter,
    ToolbarGroup,
    ToolbarItem,
    ToolbarToggleGroup,
    Tooltip,
} from '@patternfly/react-core';

import { KernelConnection, KernelManager, ServerConnection } from '@jupyterlab/services';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import {
    CheckCircleIcon,
    CodeIcon,
    CubeIcon,
    CubesIcon,
    ExclamationTriangleIcon,
    FilterIcon,
    HourglassHalfIcon,
    MigrationIcon,
    PauseIcon,
    PlusIcon,
    RebootingIcon,
    SearchIcon,
    SkullIcon,
    SpinnerIcon,
    StopCircleIcon,
    SyncIcon,
    TrashIcon,
} from '@patternfly/react-icons';

import { IKernelConnection } from '@jupyterlab/services/lib/kernel/kernel';
import { IInfoReplyMsg } from '@jupyterlab/services/lib/kernel/messages';

import {
    ConfirmationModal,
    ConfirmationWithTextInputModal,
    ExecuteCodeOnKernelModal,
    InformationModal,
} from '@app/Components/Modals';
import { DistributedJupyterKernel } from '@data/Kernel';

export interface ActionCardProps {
    onLaunchWorkloadClicked: () => void;
}

export const ActionCard: React.FunctionComponent<ActionCardProps> = (props: ActionCardProps) => {
    const [isCardExpanded, setIsCardExpanded] = React.useState(true);

    const onCardExpand = () => {
        setIsCardExpanded(!isCardExpanded);
    };

    return (
        <Card isRounded isExpanded={isCardExpanded}>
            <CardHeader
                onExpand={onCardExpand}
                toggleButtonProps={{
                    id: 'toggle-button',
                    'aria-label': 'Actions',
                    'aria-labelledby': 'titleId toggle-button',
                    'aria-expanded': isCardExpanded,
                }}
            >
                <CardTitle>
                    <Title headingLevel="h4" size="xl">
                        Actions
                    </Title>
                </CardTitle>
            </CardHeader>
            <CardExpandableContent>
                <CardBody>
                    <Button variant="primary" onClick={() => props.onLaunchWorkloadClicked()}>
                        Launch Workload
                    </Button>
                </CardBody>
            </CardExpandableContent>
        </Card>
    );
};

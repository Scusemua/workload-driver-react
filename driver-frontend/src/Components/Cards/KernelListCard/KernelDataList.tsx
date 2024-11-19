import { HeightFactorContext, KernelHeightFactorContext } from '@App/Dashboard';
import { KernelDataListCells } from '@Cards/KernelListCard/KernelDataListCells';
import { DistributedJupyterKernel, JupyterKernelReplica } from '@Data/Kernel';
import {
    DataList,
    DataListCheck,
    DataListContent,
    DataListItem,
    DataListItemRow,
    DataListToggle,
    Pagination,
    PaginationVariant,
    PerPageOptions,
} from '@patternfly/react-core';
import { useKernels } from '@Providers/KernelProvider';
import useNavigation from '@Providers/NavigationProvider';
import { KernelReplicaTable } from '@src/Components';
import { JoinPaths } from '@src/Utils/path_utils';
import { numberArrayFromRange } from '@src/Utils/utils';
import React, { useRef } from 'react';
import { useNavigate } from 'react-router-dom';

export interface KernelDataListProps {
    openMigrationModal: (kernel: DistributedJupyterKernel, replica: JupyterKernelReplica) => void;
    kernelsPerPage: number;
    perPageOption: PerPageOptions[];
    searchValue: string;
    statusSelections: string[];
    onExecuteCodeClicked: (kernel?: DistributedJupyterKernel, replicaIdx?: number | undefined) => void;
    onPingKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onInterruptKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onTerminateKernelClicked: (kernel: DistributedJupyterKernel) => void;
    onStopTrainingClicked: (kernel: DistributedJupyterKernel) => void;
    onSelectKernel: (kernelId: string) => void;
    selectedKernels: string[];
    kernelsNotClickable?: boolean;
}

export const KernelDataList: React.FunctionComponent<KernelDataListProps> = (props: KernelDataListProps) => {
    const [expandedKernels, setExpandedKernels] = React.useState<string[]>([]);

    const numKernelsCreating = useRef(0); // Used to display "pending" entries in the kernel list.

    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(props.kernelsPerPage);
    const { kernels } = useKernels(false);

    const { navigate } = useNavigation();

    const [openReplicaDropdownMenu, setOpenReplicaDropdownMenu] = React.useState<string>('');
    const [openKernelDropdownMenu, setOpenKernelDropdownMenu] = React.useState<string>('');

    const heightFactorContext: HeightFactorContext = React.useContext(KernelHeightFactorContext);

    const toggleExpandedKernel = (kernelId: string) => {
        const index = expandedKernels.indexOf(kernelId);
        const newExpanded =
            index >= 0
                ? [...expandedKernels.slice(0, index), ...expandedKernels.slice(index + 1, expandedKernels.length)]
                : [...expandedKernels, kernelId];
        setExpandedKernels(newExpanded);
    };

    const onSelectKernel = (
        _event: React.FormEvent<HTMLInputElement>,
        _checked: boolean,
        kernelId: string | undefined,
    ) => {
        if (kernelId) {
            props.onSelectKernel(kernelId);
        }
    };

    const onToggleOrSelectKernelDropdown = (kernel: DistributedJupyterKernel) => {
        if (openKernelDropdownMenu === kernel.kernelId) {
            setOpenKernelDropdownMenu('');
        } else {
            setOpenKernelDropdownMenu(kernel.kernelId || '');
            setOpenReplicaDropdownMenu('');
        }
    };

    const onFilter = (repo: DistributedJupyterKernel) => {
        // Search name with search value
        let searchValueInput: RegExp;
        try {
            searchValueInput = new RegExp(props.searchValue, 'i');
            // eslint-disable-next-line @typescript-eslint/no-unused-vars
        } catch (_err) {
            searchValueInput = new RegExp(props.searchValue.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i');
        }
        const matchesSearchValue = repo.kernelId.search(searchValueInput) >= 0;

        // Search status with status selection
        let matchesStatusValue = false;
        props.statusSelections.forEach(function (selectedStatus) {
            const match = repo.status.toLowerCase() === selectedStatus.toLowerCase();
            matchesStatusValue = matchesStatusValue || match;
        });

        return (
            (props.searchValue === '' || matchesSearchValue) &&
            (props.statusSelections.length === 0 || matchesStatusValue)
        );
    };

    const onSetPage = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPage: number) => {
        setPage(newPage);
        console.log(
            'onSetPage: Displaying workloads %d through %d.',
            perPage * (newPage - 1),
            perPage * (newPage - 1) + perPage,
        );
    };

    const onPerPageSelect = (
        _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
        newPerPage: number,
        newPage: number,
    ) => {
        setPerPage(newPerPage);
        setPage(newPage);
        heightFactorContext.setHeightFactor(Math.min(newPerPage, kernels.length));
    };

    const filteredKernels = kernels.filter(onFilter).slice(perPage * (page - 1), perPage * (page - 1) + perPage);

    const onClickKernel = (_event: React.MouseEvent | React.KeyboardEvent, id: string) => {
        const foundKernels: DistributedJupyterKernel[] | undefined = filteredKernels.filter(
            (val: DistributedJupyterKernel) => val.kernelId == id,
        );

        if (!foundKernels || foundKernels.length === 0) {
            console.warn(`Could not find kernel with ID=${id}`);
            return;
        }

        const kernel: DistributedJupyterKernel = foundKernels[0];

        console.log(`Clicked kernel ${kernel.kernelId}.`);

        navigate(['kernels', kernel.kernelId]);
    };

    const getKernelDataListRow = (kernel?: DistributedJupyterKernel, idx?: number) => {
        return (
            <DataListItem
                isExpanded={expandedKernels.includes(kernel?.kernelId || 'Pending...')}
                key={'kernel-data-row-' + (idx || -1)}
                className="kernel-list-row"
                id={kernel?.kernelId}
            >
                <DataListItemRow>
                    <DataListCheck
                        label={'kernel-' + kernel?.kernelId + '-check'}
                        aria-label={'kernel-' + kernel?.kernelId + '-check'}
                        aria-labelledby={'kernel-' + kernel?.kernelId + '-check'}
                        name={'kernel-' + kernel?.kernelId + '-check'}
                        onChange={(event: React.FormEvent<HTMLInputElement>, checked: boolean) =>
                            onSelectKernel(event, checked, kernel?.kernelId)
                        }
                        isDisabled={kernel == null}
                        defaultChecked={kernel != null && kernel.kernelId in props.selectedKernels}
                    />
                    <DataListToggle
                        onClick={() => {
                            if (kernel == null) {
                                return;
                            }

                            toggleExpandedKernel(kernel?.kernelId);
                        }}
                        isExpanded={kernel != null && expandedKernels.includes(kernel.kernelId)}
                        id={'expand-kernel-' + kernel?.kernelId + '-button'}
                        aria-controls={'expand-kernel-' + kernel?.kernelId + '-button'}
                        label={'expand-kernel-' + kernel?.kernelId + '-button'}
                        aria-label={'expand-kernel-' + kernel?.kernelId + '-button'}
                    />
                    <KernelDataListCells
                        kernel={kernel}
                        onToggleOrSelectKernelDropdown={onToggleOrSelectKernelDropdown}
                        onExecuteCodeClicked={props.onExecuteCodeClicked}
                        openKernelDropdownMenu={openKernelDropdownMenu}
                        onInterruptKernelClicked={props.onInterruptKernelClicked}
                        onPingKernelClicked={props.onPingKernelClicked}
                        onTerminateKernelClicked={props.onTerminateKernelClicked}
                        onStopTrainingClicked={props.onStopTrainingClicked}
                    />
                </DataListItemRow>
                {kernel != null && (
                    <DataListContent
                        aria-label={'kernel-' + kernel.kernelId + '-expandable-content'}
                        id={'kernel-' + kernel.kernelId + '-expandable-content'}
                        className="kernel-list-expandable-content"
                        isHidden={!expandedKernels.includes(kernel.kernelId)}
                        hasNoPadding={true}
                    >
                        <KernelReplicaTable
                            kernel={kernel}
                            openMigrationModal={props.openMigrationModal}
                            onExecuteCodeClicked={props.onExecuteCodeClicked}
                            setOpenReplicaDropdownMenu={setOpenReplicaDropdownMenu}
                            setOpenKernelDropdownMenu={setOpenKernelDropdownMenu}
                            openReplicaDropdownMenu={openReplicaDropdownMenu}
                        />
                    </DataListContent>
                )}
            </DataListItem>
        );
    };

    const pendingKernelArr = numberArrayFromRange(0, numKernelsCreating.current);

    return (
        <React.Fragment>
            <DataList
                isCompact
                aria-label="data list"
                hidden={kernels.length == 0 && pendingKernelArr.length == 0}
                onSelectDataListItem={props.kernelsNotClickable ? () => {} : onClickKernel}
            >
                {pendingKernelArr.map((_, idx) => getKernelDataListRow(undefined, idx))}
                {filteredKernels.map((kernel, idx) => getKernelDataListRow(kernel, idx + pendingKernelArr.length))}
            </DataList>
            <Pagination
                hidden={kernels.length == 0}
                isDisabled={kernels.length == 0}
                itemCount={kernels.length}
                widgetId="kernel-list-pagination"
                perPage={perPage}
                page={page}
                variant={PaginationVariant.bottom}
                perPageOptions={props.perPageOption}
                onSetPage={onSetPage}
                onPerPageSelect={onPerPageSelect}
            />
        </React.Fragment>
    );
};

import { HeightFactorContext, WorkloadsHeightFactorContext } from '@App/Dashboard';
import {
    DataList,
    DataListCell,
    DataListItem,
    DataListItemCells,
    DataListItemRow,
    Pagination,
    PaginationVariant,
    PerPageOptions,
} from '@patternfly/react-core';
import { WorkloadDataListCell } from '@src/Components/Workloads/WorkloadDataListCell';

import { Workload } from '@src/Data/Workload';
import React from 'react';

export interface IWorkloadsDataListProps {
    workloads: Workload[];
    onSelectWorkload: (event: React.MouseEvent | React.KeyboardEvent, id: string) => void;
    onClickWorkload: (workload: Workload) => void;
    onVisualizeWorkloadClicked: (workload: Workload) => void;
    workloadsPerPage?: number;
    selectedWorkloadListId: string;
    perPageOption: PerPageOptions[];
}

const WorkloadsDataList: React.FunctionComponent<IWorkloadsDataListProps> = (props: IWorkloadsDataListProps) => {
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(props.workloadsPerPage || 3);

    const heightFactorContext: HeightFactorContext = React.useContext(WorkloadsHeightFactorContext);

    const onPerPageSelect = (
        _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
        newPerPage: number,
        newPage: number,
    ) => {
        setPerPage(newPerPage);
        setPage(newPage);

        heightFactorContext.setHeightFactor(Math.min(props.workloads.length, newPerPage));
    };

    const onSetPage = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPage: number) => {
        setPage(newPage);
        // console.log(
        //     'onSetPage: Displaying workloads %d through %d.',
        //     perPage * (newPage - 1),
        //     perPage * (newPage - 1) + perPage,
        // );
    };

    return (
        <React.Fragment>
            <DataList
                isCompact
                aria-label="data list"
                selectedDataListItemId={props.selectedWorkloadListId}
                onSelectDataListItem={props.onSelectWorkload}
            >
                {props.workloads
                    .slice(perPage * (page - 1), perPage * (page - 1) + perPage)
                    .map((workload: Workload, idx: number) => (
                        <DataListItem
                            key={workload.id}
                            id={workload.id}
                            onClick={() => {
                                props.onClickWorkload(workload);
                            }}
                        >
                            <DataListItemRow>
                                <DataListItemCells
                                    dataListCells={[
                                        <DataListCell key={'workload-primary-content-' + idx} isFilled={true} width={4}>
                                            <WorkloadDataListCell
                                                onVisualizeWorkloadClicked={props.onVisualizeWorkloadClicked}
                                                workload={workload}
                                            />
                                        </DataListCell>,
                                    ]}
                                />
                            </DataListItemRow>
                        </DataListItem>
                    ))}
            </DataList>
            <Pagination
                hidden={props.workloads.length == 0}
                isDisabled={props.workloads.length == 0}
                itemCount={props.workloads.length}
                widgetId="workload-list-pagination"
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

export { WorkloadsDataList };

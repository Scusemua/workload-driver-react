import { RoundToThreeDecimalPlaces } from '@Utils/utils';
import { GpuIcon } from '@Icons/GpuIcon';
import { Flex, FlexItem, Tooltip } from '@patternfly/react-core';
import {
    CheckCircleIcon,
    CpuIcon,
    CubesIcon,
    ExclamationTriangleIcon,
    HourglassHalfIcon,
    MemoryIcon,
    RebootingIcon,
    SkullIcon,
    SpinnerIcon,
    StopCircleIcon,
} from '@patternfly/react-icons';
import { GpuIconAlt2 } from '@src/Assets/Icons';
import { DistributedJupyterKernel } from '@src/Data';
import React from 'react';

interface IKernelInfoBarProps {
    kernel?: DistributedJupyterKernel;
}

// Map from kernel status to the associated icon.
const kernelStatusIcons = {
    unknown: <ExclamationTriangleIcon />,
    starting: <SpinnerIcon className="loading-icon-spin-pulse" />,
    idle: <CheckCircleIcon />,
    busy: <HourglassHalfIcon />,
    terminating: <StopCircleIcon />,
    restarting: <RebootingIcon className="loading-icon-spin" />,
    autorestarting: <RebootingIcon className="loading-icon-spin" />,
    dead: <SkullIcon />,
};

const KernelInfoBar: React.FunctionComponent<IKernelInfoBarProps> = (props: IKernelInfoBarProps) => {
    return (
        <Flex spaceItems={{ default: 'spaceItemsMd' }} direction={{ default: 'column' }}>
            <FlexItem>
                {props.kernel != null && <p>Kernel {props.kernel.kernelId}</p>}
                {props.kernel == null && <p className="loading">Pending</p>}
            </FlexItem>
            <Flex className="kernel-list-stat-icons" spaceItems={{ default: 'spaceItemsMd' }}>
                <FlexItem>
                    <Tooltip content="Number of replicas">
                        <CubesIcon />
                    </Tooltip>
                    {props.kernel != null && props.kernel.numReplicas}
                    {props.kernel == null && 'TBD'}
                </FlexItem>
                <FlexItem>
                    {props.kernel != null && kernelStatusIcons[props.kernel.aggregateBusyStatus]}
                    {props.kernel != null && props.kernel.aggregateBusyStatus}
                    {props.kernel == null && kernelStatusIcons['starting']}
                    {props.kernel == null && 'starting'}
                </FlexItem>
                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                    <FlexItem>
                        <Tooltip content="millicpus (1/1000th of a CPU core)">
                            <CpuIcon className="node-cpu-icon" />
                        </Tooltip>
                    </FlexItem>
                    <FlexItem>
                        {(props.kernel != null &&
                            props.kernel.kernelSpec.resourceSpec.cpu != null &&
                            RoundToThreeDecimalPlaces(props.kernel.kernelSpec.resourceSpec.cpu / 1000.0)) ||
                            '0'}
                    </FlexItem>
                </Flex>
                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                    <FlexItem>
                        <Tooltip content="RAM usage limit in Gigabytes (GB)">
                            <MemoryIcon className="node-memory-icon" />
                        </Tooltip>
                    </FlexItem>
                    <FlexItem>
                        {(props.kernel != null &&
                            props.kernel.kernelSpec.resourceSpec.memory != null &&
                            RoundToThreeDecimalPlaces(props.kernel.kernelSpec.resourceSpec.memory / 1000.0)) ||
                            '0'}
                    </FlexItem>
                </Flex>
                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                    <FlexItem>
                        <Tooltip content="GPU resource usage limit">
                            <GpuIcon className="node-gpu-icon" />
                        </Tooltip>
                    </FlexItem>
                    <FlexItem>
                        {(props.kernel != null &&
                            props.kernel.kernelSpec.resourceSpec.gpu != null &&
                            props.kernel.kernelSpec.resourceSpec.gpu.toFixed(0)) ||
                            '0'}
                    </FlexItem>
                </Flex>
                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                    <FlexItem>
                        <Tooltip content="VRAM resource usage limit">
                            <GpuIconAlt2 className="node-gpu-icon" />
                        </Tooltip>
                    </FlexItem>
                    <FlexItem>
                        {(props.kernel != null &&
                            props.kernel.kernelSpec.resourceSpec.vram != null &&
                            props.kernel.kernelSpec.resourceSpec.vram.toFixed(0)) ||
                            '0'}
                    </FlexItem>
                </Flex>
            </Flex>
        </Flex>
    );
};

export default KernelInfoBar;

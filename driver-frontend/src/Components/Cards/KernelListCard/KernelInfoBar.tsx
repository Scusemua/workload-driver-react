import { GpuIcon } from '@Icons/GpuIcon';
import { Flex, FlexItem, Icon, Text, Tooltip } from '@patternfly/react-core';
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
import { RoundToThreeDecimalPlaces } from '@Utils/utils';
import React, { ReactElement } from 'react'; // Map from kernel status to the associated icon.

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

interface IKernelInfoIconsProps {
    kernel?: DistributedJupyterKernel;
    iconSizes?: 'sm' | 'md' | 'lg' | 'xl';
    iconSpacingOverride?:
        | 'spaceItemsNone'
        | 'spaceItemsXs'
        | 'spaceItemsSm'
        | 'spaceItemsMd'
        | 'spaceItemsLg'
        | 'spaceItemsXl'
        | 'spaceItems2xl'
        | 'spaceItems3xl'
        | 'spaceItems4xl';
}

export const KernelInfoIcons: React.FunctionComponent<IKernelInfoIconsProps> = (props: IKernelInfoIconsProps) => {
    const getLabelFontSize = () => {
        if (props.iconSizes == 'xl') {
            return 27;
        } else if (props.iconSizes == 'lg') {
            return 23;
        } else if (props.iconSizes == 'md') {
            return 19;
        }

        return 15;
    };

    /**
     * Return the amount of spacing that there should be between each icon and its label.
     */
    const getIconAndLabelSpacing = () => {
        if (props.iconSizes == 'xl') {
            return 'spaceItemsLg';
        } else if (props.iconSizes == 'lg') {
            return 'spaceItemsMd';
        } else if (props.iconSizes == 'md') {
            return 'spaceItemsSm';
        }

        return 'spaceItemsSm';
    };

    /**
     * Return the elements required to render and icon and its label.
     * @param icon icon to be shown
     * @param label text for the label
     * @param tooltipContent if given, icon will be wrapped in a tooltip with this as the hint
     */
    const getIconAndLabel = (icon: ReactElement, label: string | number | undefined, tooltipContent?: string) => {
        if (tooltipContent) {
            return (
                <Flex direction={{ default: 'row' }} spaceItems={{ default: getIconAndLabelSpacing() }}>
                    <FlexItem>
                        <Tooltip content={tooltipContent}>
                            <Icon size={props.iconSizes}>{icon}</Icon>
                        </Tooltip>
                    </FlexItem>
                    <FlexItem>
                        <Text component={'p'} style={{ fontSize: getLabelFontSize() }}>
                            {label}
                        </Text>
                    </FlexItem>
                </Flex>
            );
        } else {
            return (
                <Flex direction={{ default: 'row' }} spaceItems={{ default: getIconAndLabelSpacing() }}>
                    <FlexItem>
                        <Icon size={props.iconSizes}>{icon}</Icon>
                    </FlexItem>
                    <FlexItem>
                        <Text component={'p'} style={{ fontSize: getLabelFontSize() }}>
                            {label}
                        </Text>
                    </FlexItem>
                </Flex>
            );
        }
    };

    return (
        <Flex className="kernel-list-stat-icons" spaceItems={{ default: props.iconSpacingOverride || 'spaceItemsLg' }}>
            {getIconAndLabel(<CubesIcon />, props.kernel ? props.kernel.numReplicas : 'TBD')}
            {getIconAndLabel(
                props.kernel ? kernelStatusIcons[props.kernel.aggregateBusyStatus] : kernelStatusIcons['starting'],
                props.kernel ? props.kernel.aggregateBusyStatus : 'starting',
            )}
            {getIconAndLabel(
                <CpuIcon className="node-cpu-icon" />,
                (props.kernel != null &&
                    props.kernel.kernelSpec.resourceSpec.cpu != null &&
                    RoundToThreeDecimalPlaces(props.kernel.kernelSpec.resourceSpec.cpu / 1000.0)) ||
                    '0',
                'millicpus (1/1000th of a CPU core)',
            )}
            {getIconAndLabel(
                <MemoryIcon className="node-memory-icon" />,
                (props.kernel != null &&
                    props.kernel.kernelSpec.resourceSpec.memory != null &&
                    RoundToThreeDecimalPlaces(props.kernel.kernelSpec.resourceSpec.memory / 1000.0)) ||
                    '0',
                'RAM usage limit in Gigabytes (GB)',
            )}
            {getIconAndLabel(
                <GpuIcon className="node-gpu-icon" />,
                (props.kernel != null &&
                    props.kernel.kernelSpec.resourceSpec.gpu != null &&
                    props.kernel.kernelSpec.resourceSpec.gpu.toFixed(0)) ||
                    '0',
                'GPU resource usage limit',
            )}
            {getIconAndLabel(
                <GpuIconAlt2 className="node-gpu-icon" />,
                (props.kernel != null &&
                    props.kernel.kernelSpec.resourceSpec.vram != null &&
                    props.kernel.kernelSpec.resourceSpec.vram.toFixed(0)) ||
                    '0',
                'VRAM resource usage limit in GB',
            )}
        </Flex>
    );
};

interface IKernelInfoBarProps {
    kernel?: DistributedJupyterKernel;
    iconSizes?: 'sm' | 'md' | 'lg' | 'xl';
}

export const KernelInfoBar: React.FunctionComponent<IKernelInfoBarProps> = (props: IKernelInfoBarProps) => {
    return (
        <Flex spaceItems={{ default: 'spaceItemsMd' }} direction={{ default: 'column' }}>
            <FlexItem>
                {props.kernel != null && <p>Kernel {props.kernel.kernelId}</p>}
                {props.kernel == null && <p className="loading">Pending</p>}
            </FlexItem>
            <KernelInfoIcons kernel={props.kernel} iconSizes={props.iconSizes} iconSpacingOverride={'spaceItemsMd'} />
        </Flex>
    );
};

export default KernelInfoBar;

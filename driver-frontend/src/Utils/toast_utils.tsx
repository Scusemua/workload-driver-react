import { Alert, AlertActionCloseButton, Flex, FlexItem } from '@patternfly/react-core';
import { SpinnerIcon } from '@patternfly/react-icons';
import { RoundToThreeDecimalPlaces } from '@Utils/utils';
import React, { ReactElement, ReactNode } from 'react';
import { Toast, toast } from 'react-hot-toast';
import { DefaultToastOptions, Renderable } from 'react-hot-toast/src/core/types';

export function GetHttpErrorMessage(res: Response, reason: string | any): string {
    return `HTTP ${res.status} - ${res.statusText}: ${JSON.stringify(reason)}`;
}

/**
 * Return a <div> containing a <Flex> to be used as a toast notification.
 * @param header Name or title of the error
 * @param body Error message
 * @param variant The variant of alert to display
 * @param dismissToast Called when the toast should be dismissed.
 * @param timeout optional timeout for the alert
 * @param customIcon custom icon to display, optional
 */
export function GetToastContentWithHeaderAndBody(
    header: string,
    body: string | ReactElement | (string | ReactElement)[] | undefined | null,
    variant: 'danger' | 'warning' | 'success' | 'info' | 'custom',
    dismissToast: () => void,
    timeout: number | boolean = false,
    customIcon?: ReactNode,
): ReactElement {
    const getAlertContent = () => {
        if (!body) {
            return <React.Fragment />;
        }

        if (typeof body === 'string') {
            return <p>{body}</p>;
        }

        if (React.isValidElement(body)) {
            return body as ReactElement;
        }

        if (Array.isArray(body)) {
            return (
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
                    {(body as (string | ReactElement)[]).map((elem: string | ReactElement, index: number) => {
                        if (typeof elem === 'string') {
                            return (
                                <FlexItem key={`toast-content-row-${index}`}>
                                    <p>{elem as string}</p>
                                </FlexItem>
                            );
                        } else {
                            return <FlexItem key={`toast-content-row-${index}`}>{elem as ReactElement}</FlexItem>;
                        }
                    })}
                </Flex>
            );
        }

        throw new Error(`Unexpected type for body parameter (${typeof body}): ${body}`);
    };

    return (
        <Alert
            isInline
            variant={variant}
            title={header}
            timeoutAnimation={timeout ? 30000 : undefined}
            timeout={timeout}
            onTimeout={() => {
                console.log('Alert has timed-out');
                dismissToast();
            }}
            customIcon={customIcon}
            actionClose={<AlertActionCloseButton onClose={() => dismissToast()} />}
        >
            {getAlertContent()}
        </Alert>
    );
}

/**
 * Simple, default dismiss function for dismissing a toast. Can be used with the other helpers, such
 * as GetToastContentWithHeaderAndBody.
 *
 * DefaultDismiss returns a function/callable that can be used to dismiss the specified toast.
 *
 * @param toastId the ID of the toast that is to be dismissed when the function is called
 */
export function DefaultDismiss(toastId?: string): () => void {
    return () => {
        if (toastId) {
            toast.dismiss(toastId);
        }
    };
}

export async function ToastPromise<T>(
    promise: () => Promise<T>,
    loading: (t: Toast) => Renderable,
    success: (t: Toast, result: T, latencyMilliseconds: number) => ReactElement | string | null,
    error: (t: Toast, e: any) => ReactElement | string | null,
    opts?: DefaultToastOptions,
): Promise<T | null> {
    const toastId: string = toast.custom((t: Toast) => loading(t), { ...opts, ...opts?.loading });
    const start: number = performance.now();

    return await promise()
        .then((result: T) => {
            const latencyMilliseconds: number = performance.now() - start;
            toast.custom((t: Toast) => success(t, result, latencyMilliseconds), {
                id: toastId,
                ...opts,
                ...opts?.success,
            });

            return result;
        })
        .catch((e) => {
            toast.custom((t: Toast) => error(t, e), {
                id: toastId,
                ...opts,
                ...opts?.success,
            });

            return null;
        });
}

/**
 * Basically an implementation of toast.promise(), but this (a) actually works, and (b) is targeted specifically
 * for refreshing a remote resource.
 *
 * @param refreshFunc the function (usually an SWR hook/trigger/mutator) to perform the refresh. Must be async.
 * @param loadingMessage message to display in toast notification while refreshing.
 * @param errorMessage message to display in toast notification if an error occurs.
 * @param successMessage message to display in toast notification upon successful refresh. this will have " in X milliseconds"
 * appended to the end of it before being displayed.
 */
export function ToastRefresh<T>(
    refreshFunc: () => Promise<T>,
    loadingMessage: string,
    errorMessage: string,
    successMessage: string,
) {
    const toastId: string = toast.custom((t: Toast) => {
        return (
            <Alert
                isInline
                variant={'info'}
                title={loadingMessage}
                onTimeout={() => toast.dismiss(t.id)}
                customIcon={<SpinnerIcon className={'loading-icon-spin-pulse'} />}
                actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
            />
        );
    });

    const start: number = performance.now();
    refreshFunc()
        .then(() => {
            const latencyMs: number = RoundToThreeDecimalPlaces(performance.now() - start);
            toast.custom(
                (t) => {
                    return (
                        <Alert
                            isInline
                            variant={'success'}
                            title={successMessage + ` in ${latencyMs} milliseconds.`}
                            onTimeout={() => toast.dismiss(t.id)}
                            timeout={10000}
                            timeoutAnimation={20000}
                            actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
                        />
                    );
                },
                {
                    id: toastId,
                },
            );
        })
        .catch((err: Error) => {
            console.error(`ToastRefresh ERROR: ${err}`);
            const latencyMs: number = RoundToThreeDecimalPlaces(performance.now() - start);
            toast.custom(
                (t) => {
                    return (
                        <Alert
                            isInline
                            variant={'danger'}
                            title={errorMessage + ` after ${latencyMs} milliseconds`}
                            onTimeout={() => toast.dismiss(t.id)}
                            timeout={30000}
                            timeoutAnimation={60000}
                            actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
                        >
                            {`${err.name}: ${err.message}`}
                        </Alert>
                    );
                },
                { id: toastId },
            );
        });
}

export async function ToastFetch(
    loadingMessage: string,
    successToast: (toastId: string) => ReactElement,
    errorToast: (resp: Response, reason: string, toastId: string) => ReactElement,
    endpoint: string,
    requestOptions: RequestInit | undefined,
) {
    const toastId: string = toast.custom((t: Toast) => (
        <Alert
            isInline
            variant={'info'}
            title={loadingMessage}
            onTimeout={() => toast.dismiss(t.id)}
            customIcon={<SpinnerIcon className={'loading-icon-spin-pulse'} />}
            actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
        />
    ));
    await fetch(endpoint, requestOptions).then((res) => {
        if (!res.ok || res.status >= 300) {
            res.json().then((reason) => {
                console.error(`HTTP ${res.status} - ${res.statusText}: ${JSON.stringify(reason)}`);
                toast.custom(errorToast(res, reason, toastId), {
                    id: toastId,
                    duration: 10000,
                    style: { maxWidth: 600 },
                });
            });
        } else {
            toast.custom(successToast(toastId), { id: toastId, duration: 7500, style: { maxWidth: 600 } });
        }
    });
}

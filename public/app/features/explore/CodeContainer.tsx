import React, { PureComponent } from 'react';
import { connect, ConnectedProps } from 'react-redux';
import { splitOpen } from './state/main';
import { ExploreId, ExploreItemState } from 'app/types/explore';
import { StoreState } from 'app/types';

interface CodeContainerProps {
  exploreId: ExploreId;
}

function mapStateToProps(state: StoreState, { exploreId }: CodeContainerProps) {
  const explore = state.explore;
  // @ts-ignore
  const item: ExploreItemState = explore[exploreId];
  // const { loading: loadingInState, tableResult, range } = item;
  // const loading = tableResult && tableResult.length > 0 ? false : loadingInState;
  // return { loading, tableResult, range };
  return {};
}

const mapDispatchToProps = {
  splitOpen,
};

const connector = connect(mapStateToProps, mapDispatchToProps);

type Props = CodeContainerProps & ConnectedProps<typeof connector>;

export class CodeContainer extends PureComponent<Props> {
  render() {
    return <div>THIS IS THE CODE!!!</div>;
  }
}

export default connector(CodeContainer);

<script setup lang="ts" generic="T extends RowData">
import { FlexRender, type RowData, type Table } from '@tanstack/vue-table'
import { ListX } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

import { Card } from '@/components/ui/card'
import {
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	Table as TableRoot,
	TableRow,
} from '@/components/ui/table'
import { useIsMobile } from '@/composables/use-is-mobile'
import ShadcnLayout from '@/layout/shadcn-layout.vue'

defineProps<{
	table: Table<T>
	isLoading: boolean
	hideHeader?: boolean
}>()

const { isDesktop } = useIsMobile()
const { t } = useI18n()
</script>

<template>
	<ShadcnLayout v-if="isDesktop">
		<TableRoot>
			<TableHeader v-if="!hideHeader">
				<TableRow v-for="headerGroup in table.getHeaderGroups()" :key="headerGroup.id" class="border-b">
					<TableHead v-for="header in headerGroup.headers" :key="header.id" :style="{ width: `${header.getSize()}%` }">
						<FlexRender
							v-if="!header.isPlaceholder"
							:render="header.column.columnDef.header"
							:props="header.getContext()"
						/>
					</TableHead>
				</TableRow>
			</TableHeader>
			<TableBody :class="[isLoading ? 'animate-pulse' : '']">
				<template v-if="table.getRowModel().rows?.length">
					<TableRow
						v-for="row in table.getRowModel().rows" :key="row.id"
						:data-state="row.getIsSelected() ? 'selected' : undefined" class="border-b"
					>
						<TableCell
							v-for="cell in row.getVisibleCells()"
							:key="cell.id"
							class="md:break-all"
							:class="{
								'cursor-pointer': row.getCanExpand(),
							}"
							@click="() => {
								if (row.getCanExpand()) {
									row.getToggleExpandedHandler()()
								}
							}"
						>
							<FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
						</TableCell>
					</TableRow>
				</template>
				<template v-else>
					<TableRow>
						<TableCell :colSpan="table.getAllColumns().length" class="h-24 text-center">
							<slot name="empty-message">
								<div class="flex items-center flex-col justify-center">
									<ListX class="size-12" />
									<span class="font-medium text-2xl">{{ t('sharedTexts.noData') }}</span>
								</div>
							</slot>
						</TableCell>
					</TableRow>
				</template>
			</TableBody>
		</TableRoot>
	</ShadcnLayout>

	<div v-else class="grid grid-cols-1 gap-4">
		<Card v-for="row in table.getRowModel().rows" :key="row.id">
			<div
				v-for="cell in row.getVisibleCells()"
				:key="cell.id"
			>
				<div v-if="row.getCanExpand()" class="px-2 my-2 cursor-pointer">
					<FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" @click="() => row.getToggleExpandedHandler()()" />
				</div>

				<div v-else-if="cell.column.id !== 'actions'" class="px-4 py-2 border-b-2">
					<FlexRender :render="cell.column.columnDef.header" class="text-sm text-zinc-400/80" />
					<FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
				</div>

				<div v-else class="flex h-auto py-2 px-2 justify-end">
					<FlexRender
						:render="cell.column.columnDef.cell"
						:props="cell.getContext()"
					/>
				</div>
			</div>
		</Card>
	</div>
</template>

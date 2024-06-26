#include <stdlib.h>
#include <stdio.h>
#include <errno.h>
#include <unistd.h>
#include <string.h>
#include <signal.h>
#include <stdint.h>
#include <stdbool.h>
#include <sys/mman.h>
#include <sys/resource.h>
#include <math.h>
#include "dsm.h"
#include "_cgo_export.h"

#define ROW_A 64
#define COL_A 64
#define ROW_B 64
#define COL_B 64

int* matrixA;
int* matrixB;
int* matrixC;

void setup_matmul(int num_pages, int index, int total_servers) {
    setup_handler();

    // map all pages as PROT_READ | PROT_WRITE
    if (index == 0) {
        printf("Mapping all pages as PROT_READ | PROT_WRITE\n");
        create_pages(num_pages);

        matrixA = (int *)p;
        matrixB = (int *)(p + PAGE_SIZE * 4);
        matrixC = (int *)(p + PAGE_SIZE * 8);
        for (int i = 0; i < ROW_A * COL_A; i++) {
            matrixA[i] = 1;
        }
        for (int i = 0; i < ROW_B * COL_B; i++) {
            matrixB[i] = 1;
        }

        for (int i = 0; i < ROW_A * COL_B; i++) {
            matrixC[i] = 0;
        }
        printf("Matrix A:\n");
        print_matrix(ROW_A, COL_A, matrixA);
        printf("Matrix B:\n");
        print_matrix(ROW_B, COL_B, matrixB);
    } else {
        printf("Mapping all pages as PROT_NONE\n");
        create_pages(num_pages);
        matrixA = (int *)p;
        matrixB = (int *)(p + PAGE_SIZE * 4);
        matrixC = (int *)(p + PAGE_SIZE * 8);
        if (p == MAP_FAILED) {
            fprintf(stderr, "Couldn't mmap memory; %s\n", strerror(errno));
            exit(EXIT_FAILURE);
        }
    }
}

void print_matrix(int rows, int cols, int *matrix) {
    int i, j;
    for (i = 0; i < rows; i++) {
        for (j = 0; j < cols; j++) {
            printf("%d\t", matrix[i * cols + j]);
        }
        printf("\n");
    }
}

void multiply_matrices(int index, int total_servers) {
    int i, j, k;
    int start = (int)floor((index / (double)total_servers) * ROW_A);
    int end = (int)floor(((index + 1) / (double)total_servers) * ROW_A);

    for (i =start; i < end; i++) {
        for (j = 0; j < COL_B; j++) {
            int val = 0;
            for (k = 0; k < COL_A; k++) {
                val += matrixA[i * COL_A + k] * matrixB[k * COL_B + j];
            }
            matrixC[i * COL_B + j] = val;
        }
    }
    printf("Matrix C:\n");
    print_matrix(ROW_A, COL_B, matrixC);
}